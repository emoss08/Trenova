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

func (sm *StopStateMachine) CanTransition(event TransitionEvent) bool {
	currentState := sm.stop.Status

	// Define valid transition based on current state and event
	validTransitions := map[shipment.StopStatus]map[TransitionEvent]bool{
		shipment.StopStatusNew: {
			EventStopArrived:  true,
			EventStopCanceled: true,
			EventStopDeparted: true,
		},
		shipment.StopStatusInTransit: {
			EventStopDeparted: true,
			EventStopCanceled: true,
		},
		shipment.StopStatusCompleted: {
			EventStopCanceled: true,
		},
		// Terminal State - no transitions allowed
		shipment.StopStatusCanceled: {},
	}

	if transitions, exists := validTransitions[currentState]; exists {
		return transitions[event]
	}

	log.Debug().
		Str("stopID", sm.stop.ID.String()).
		Str("event", event.EventType()).
		Str("currentState", string(currentState)).
		Msg("stop transition not allowed")

	return false
}

func (sm *StopStateMachine) Transition(event TransitionEvent) error {
	if !sm.CanTransition(event) {
		return newTransitionError(
			string(sm.stop.Status),
			event,
			"transition not allowed",
		)
	}

	var newStatus shipment.StopStatus

	switch event {
	case EventStopArrived:
		newStatus = shipment.StopStatusInTransit
	case EventStopDeparted:
		newStatus = shipment.StopStatusCompleted
	case EventStopCanceled:
		newStatus = shipment.StopStatusCanceled
	default:
		return newTransitionError(
			string(sm.stop.Status),
			event,
			"unsupported event",
		)
	}

	// Update the stop status
	sm.stop.Status = newStatus

	return nil
}

func (sm *StopStateMachine) IsInTerminalState() bool {
	return sm.stop.Status == shipment.StopStatusCanceled
}
