package statemachine

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/rs/zerolog/log"
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

func (sm *MoveStateMachine) CanTransition(ctx context.Context, event TransitionEvent) bool {
	currentState := sm.move.Status

	// Define valid transitions based on current state and event
	validTransitions := map[shipment.MoveStatus]map[TransitionEvent]bool{
		shipment.MoveStatusNew: {
			EventMoveAssigned:  true,
			EventMoveCanceled:  true,
			EventMoveStarted:   true,
			EventMoveCompleted: true,
		},
		shipment.MoveStatusAssigned: {
			EventMoveStarted:  true,
			EventMoveCanceled: true,
		},
		shipment.MoveStatusInTransit: {
			EventMoveCompleted: true,
			EventMoveCanceled:  true,
		},
		shipment.MoveStatusCompleted: {
			EventMoveCanceled: true,
		},
		// Terminal State - no transitions sllowed
		shipment.MoveStatusCanceled: {},
	}

	if transitions, exists := validTransitions[currentState]; exists {
		return transitions[event]
	}

	log.Info().
		Str("moveID", sm.move.ID.String()).
		Str("event", string(event)).
		Str("currentState", string(currentState)).
		Msg("move transition not allowed")

	return false
}

func (sm *MoveStateMachine) Transition(ctx context.Context, event TransitionEvent) error {
	if !sm.CanTransition(ctx, event) {
		return newTransitionError(
			string(sm.move.Status),
			"<unknown>",
			event,
			"transition not allowed",
		)
	}

	var newStatus shipment.MoveStatus

	switch event {
	case EventMoveAssigned:
		newStatus = shipment.MoveStatusAssigned
	case EventMoveStarted:
		newStatus = shipment.MoveStatusInTransit
	case EventMoveCompleted:
		newStatus = shipment.MoveStatusCompleted
	case EventMoveCanceled:
		newStatus = shipment.MoveStatusCanceled
	default:
		return newTransitionError(
			string(sm.move.Status),
			"<unknown>",
			event,
			"unsupport event",
		)
	}

	// Update the move status
	sm.move.Status = newStatus

	return nil
}

// IsInTerminalState checks if the current state is terminal
func (sm *MoveStateMachine) IsInTerminalState() bool {
	return sm.move.Status == shipment.MoveStatusCanceled
}
