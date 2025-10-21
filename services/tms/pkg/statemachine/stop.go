package statemachine

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
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

var validStopTransitions = map[shipment.StopStatus]map[TransitionEvent]shipment.StopStatus{ //nolint:exhaustive // we don't need to add StopStatusCanceled to the transitions
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

	return NewTransitionError(
		string(currentState),
		event,
		"transition not allowed or event not defined for current state",
	)
}

func (sm *StopStateMachine) IsInTerminalState() bool {
	return sm.stop.Status == shipment.StopStatusCanceled
}
