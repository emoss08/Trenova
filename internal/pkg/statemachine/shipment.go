package statemachine

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type ShipmentStateMachine struct {
	shipment *shipment.Shipment
}

func NewShipmentStateMachine(shp *shipment.Shipment) StateMachine {
	return &ShipmentStateMachine{
		shipment: shp,
	}
}

func (sm *ShipmentStateMachine) CurrentState() string {
	return string(sm.shipment.Status)
}

func (sm *ShipmentStateMachine) CanTransition(ctx context.Context, event TransitionEvent) bool {
	currState := sm.shipment.Status

	validTransitions := map[shipment.Status]map[TransitionEvent]bool{
		shipment.StatusNew: {
			EventShipmentPartiallyAssigned: true,
			EventShipmentAssigned:          true,
			EventShipmentInTransit:         true,
			EventShipmentPartialCompleted:  true,
			EventShipmentCompleted:         true, // * It's possible to complete a shipment during it's creation.
			EventShipmentCanceled:          true,
		},
		shipment.StatusPartiallyAssigned: {
			EventShipmentAssigned:  true,
			EventShipmentInTransit: true,
			EventShipmentCanceled:  true,
		},
		shipment.StatusAssigned: {
			EventShipmentInTransit: true,
			EventShipmentCanceled:  true,
		},
		shipment.StatusInTransit: {
			EventShipmentDelayed:          true,
			EventShipmentPartialCompleted: true,
			EventShipmentCompleted:        true,
			EventShipmentCanceled:         true,
		},
		shipment.StatusDelayed: {
			EventShipmentInTransit:        true,
			EventShipmentPartialCompleted: true,
			EventShipmentCompleted:        true,
			EventShipmentCanceled:         true,
		},
		shipment.StatusPartiallyCompleted: {
			EventShipmentCompleted: true,
			EventShipmentCanceled:  true,
		},
		shipment.StatusCompleted: {
			EventShipmentCanceled: true,
		},

		// terminal state - do not allow transitions
		shipment.StatusCanceled: {},
		shipment.StatusBilled:   {},
	}

	if transitions, exists := validTransitions[currState]; exists {
		return transitions[event]
	}

	return false
}

func (sm *ShipmentStateMachine) Transition(ctx context.Context, event TransitionEvent) error {
	if !sm.CanTransition(ctx, event) {
		return newTransitionError(
			string(sm.shipment.Status),
			"<unknown>",
			event,
			"transition not allowed",
		)
	}

	// Apply the transition
	var newStatus shipment.Status

	switch event {
	case EventShipmentPartiallyAssigned:
		newStatus = shipment.StatusPartiallyAssigned
	case EventShipmentAssigned:
		newStatus = shipment.StatusAssigned
	case EventShipmentInTransit:
		newStatus = shipment.StatusInTransit
	case EventShipmentDelayed:
		newStatus = shipment.StatusDelayed
	case EventShipmentPartialCompleted:
		newStatus = shipment.StatusPartiallyCompleted
	case EventShipmentCompleted:
		newStatus = shipment.StatusCompleted
	case EventShipmentCanceled:
		newStatus = shipment.StatusCanceled
	default:
		return newTransitionError(
			string(sm.shipment.Status),
			"<unknown>",
			event,
			"unsupported event",
		)
	}

	// Update the shipment status
	sm.shipment.Status = newStatus

	return nil
}

func (sm *ShipmentStateMachine) IsInTerminalState() bool {
	return sm.shipment.StatusEquals(shipment.StatusBilled) || sm.shipment.StatusEquals(shipment.StatusCanceled)
}
