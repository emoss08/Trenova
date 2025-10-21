package statemachine

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type MoveStateMachine struct {
	move *shipment.ShipmentMove
}

func NewMoveStateMachine(move *shipment.ShipmentMove) StateMachine {
	return &MoveStateMachine{
		move: move,
	}
}

func (sm *MoveStateMachine) CurrentState() string {
	return string(sm.move.Status)
}

//nolint:exhaustive // No need to include the terminal states
var validMoveTransitions = map[shipment.MoveStatus]map[TransitionEvent]shipment.MoveStatus{
	shipment.MoveStatusNew: {
		EventMoveAssigned:  shipment.MoveStatusAssigned,
		EventMoveCanceled:  shipment.MoveStatusCanceled,
		EventMoveStarted:   shipment.MoveStatusInTransit, // New can go directly to InTransit if started
		EventMoveCompleted: shipment.MoveStatusCompleted, // New can go directly to Completed
	},
	shipment.MoveStatusAssigned: {
		EventMoveStarted:   shipment.MoveStatusInTransit,
		EventMoveCanceled:  shipment.MoveStatusCanceled,
		EventMoveCompleted: shipment.MoveStatusCompleted, // Assigned can go to Completed
	},
	shipment.MoveStatusInTransit: {
		EventMoveCompleted: shipment.MoveStatusCompleted,
		EventMoveCanceled:  shipment.MoveStatusCanceled,
	},
	shipment.MoveStatusCompleted: {
		EventMoveCanceled: shipment.MoveStatusCanceled, // e.g., if a completed move needs to be voided
	},
	// shipment.MoveStatusCanceled is a terminal state, no transitions out.
}

func (sm *MoveStateMachine) CanTransition(event TransitionEvent) bool {
	currentState := sm.move.Status
	if transitions, ok := validMoveTransitions[currentState]; ok {
		if _, eventAllowed := transitions[event]; eventAllowed {
			return true
		}
	}

	return false
}

func (sm *MoveStateMachine) Transition(event TransitionEvent) error {
	currentState := sm.move.Status
	if transitions, ok := validMoveTransitions[currentState]; ok {
		if newStatus, eventAllowed := transitions[event]; eventAllowed {
			sm.move.Status = newStatus
			return nil
		}
	}

	return NewTransitionError(
		string(currentState),
		event,
		"transition not allowed or event not defined for current state",
	)
}

func (sm *MoveStateMachine) IsInTerminalState() bool {
	// ! A move is terminal if it's Canceled or if there are no further transitions defined from its current state.
	// ! For simplicity, we'll stick to just Canceled as explicitly terminal for now.
	return sm.move.Status == shipment.MoveStatusCanceled
}
