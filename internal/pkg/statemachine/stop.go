package statemachine

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/rs/zerolog/log"
)

type StopStateMachine struct {
	stop *shipment.Stop
}

func NewStopStateMachine(stop *shipment.Stop) StateMachine {
	return &StopStateMachine{
		stop: stop,
	}
}

func (sm *StopStateMachine) CurrentState() string {
	return string(sm.stop.Status)
}

// validStopTransitions defines the allowed state transitions and the target state for each.
// Format: map[currentState]map[triggeringEvent]targetState
//
//nolint:exhaustive // No need to include the terminal states
var validStopTransitions = map[shipment.StopStatus]map[TransitionEvent]shipment.StopStatus{
	shipment.StopStatusNew: {
		EventStopArrived:  shipment.StopStatusInTransit,
		EventStopCanceled: shipment.StopStatusCanceled,
		EventStopDeparted: shipment.StopStatusCompleted, // New can go directly to Completed if departed (e.g. quick load/unload)
	},
	shipment.StopStatusInTransit: {
		EventStopDeparted: shipment.StopStatusCompleted,
		EventStopCanceled: shipment.StopStatusCanceled,
	},
	shipment.StopStatusCompleted: {
		EventStopCanceled: shipment.StopStatusCanceled, // e.g., if a completed stop needs to be voided
	},
	// shipment.StopStatusCanceled is a terminal state, no transitions out.
}

func (sm *StopStateMachine) CanTransition(event TransitionEvent) bool {
	currentState := sm.stop.Status
	if transitions, ok := validStopTransitions[currentState]; ok {
		if _, eventAllowed := transitions[event]; eventAllowed {
			return true
		}
	}

	log.Debug(). // Changed from Info to Debug as this might be noisy otherwise
			Str("stopID", sm.stop.ID.String()).
			Str("event", event.EventType()).
			Str("currentState", string(currentState)).
			Msg("stop transition not allowed")

	return false
}

func (sm *StopStateMachine) Transition(event TransitionEvent) error {
	currentState := sm.stop.Status
	if transitions, ok := validStopTransitions[currentState]; ok {
		if newStatus, eventAllowed := transitions[event]; eventAllowed {
			sm.stop.Status = newStatus
			return nil
		}
	}

	// If we reach here, the transition is not allowed.
	return newTransitionError(
		string(currentState),
		event,
		"transition not allowed or event not defined for current state",
	)
}

func (sm *StopStateMachine) IsInTerminalState() bool {
	return sm.stop.Status == shipment.StopStatusCanceled
}
