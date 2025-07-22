package statemachine

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/rs/zerolog/log"
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

// validShipmentTransitions defines the allowed state transitions and the target state for each.
// Format: map[currentState]map[triggeringEvent]targetState
//
//nolint:exhaustive // No need to include the terminal states
var validShipmentTransitions = map[shipment.Status]map[TransitionEvent]shipment.Status{
	shipment.StatusNew: {
		EventShipmentPartiallyAssigned: shipment.StatusPartiallyAssigned,
		EventShipmentAssigned:          shipment.StatusAssigned,
		EventShipmentInTransit:         shipment.StatusInTransit,
		EventShipmentPartialCompleted:  shipment.StatusPartiallyCompleted,
		EventShipmentCompleted:         shipment.StatusCompleted,
		EventShipmentCanceled:          shipment.StatusCanceled,
	},
	shipment.StatusPartiallyAssigned: {
		EventShipmentAssigned:  shipment.StatusAssigned,
		EventShipmentInTransit: shipment.StatusInTransit, // * Can go to InTransit if remaining moves start
		EventShipmentCanceled:  shipment.StatusCanceled,
	},
	shipment.StatusAssigned: {
		EventShipmentInTransit: shipment.StatusInTransit,
		EventShipmentCompleted: shipment.StatusCompleted, // If all moves are somehow completed while shipment is Assigned
		EventShipmentCanceled:  shipment.StatusCanceled,
	},
	shipment.StatusInTransit: {
		EventShipmentDelayed:          shipment.StatusDelayed,
		EventShipmentPartialCompleted: shipment.StatusPartiallyCompleted,
		EventShipmentCompleted:        shipment.StatusCompleted,
		EventShipmentCanceled:         shipment.StatusCanceled,
	},
	shipment.StatusDelayed: {
		EventShipmentInTransit:        shipment.StatusInTransit,
		EventShipmentPartialCompleted: shipment.StatusPartiallyCompleted,
		EventShipmentCompleted:        shipment.StatusCompleted,
		EventShipmentCanceled:         shipment.StatusCanceled,
	},
	shipment.StatusPartiallyCompleted: {
		EventShipmentCompleted: shipment.StatusCompleted,
		EventShipmentCanceled:  shipment.StatusCanceled,
	},
	shipment.StatusCompleted: {
		EventShipmentReadyToBill: shipment.StatusReadyToBill,
		EventShipmentCanceled:    shipment.StatusCanceled, // e.g., if a completed shipment needs to be voided before billing
	},
	shipment.StatusReadyToBill: {
		EventShipmentBilled:         shipment.StatusBilled,
		EventShipmentReviewRequired: shipment.StatusReviewRequired,
		EventShipmentCanceled:       shipment.StatusCanceled,
	},
	shipment.StatusReviewRequired: {
		EventShipmentBilled:   shipment.StatusBilled,
		EventShipmentCanceled: shipment.StatusCanceled,
	},
	// shipment.StatusBilled and shipment.StatusCanceled are terminal states.
}

func (sm *ShipmentStateMachine) CanTransition(event TransitionEvent) bool {
	currentState := sm.shipment.Status
	if transitions, ok := validShipmentTransitions[currentState]; ok {
		if _, eventAllowed := transitions[event]; eventAllowed {
			return true
		}
	}

	log.Debug().
		Str("shipmentID", sm.shipment.ID.String()).
		Str("event", event.EventType()).
		Str("currentState", string(currentState)).
		Msg("shipment transition not allowed")

	return false
}

func (sm *ShipmentStateMachine) Transition(event TransitionEvent) error {
	currentState := sm.shipment.Status
	if transitions, ok := validShipmentTransitions[currentState]; ok {
		if newStatus, eventAllowed := transitions[event]; eventAllowed {
			sm.shipment.Status = newStatus
			return nil
		}
	}

	return newTransitionError(
		string(currentState),
		event,
		"transition not allowed or event not defined for current state",
	)
}

func (sm *ShipmentStateMachine) IsInTerminalState() bool {
	// A shipment is terminal if it's Billed or Canceled.
	// Could also check if validShipmentTransitions[sm.shipment.Status] is empty.
	return sm.shipment.StatusEquals(shipment.StatusBilled) ||
		sm.shipment.StatusEquals(shipment.StatusCanceled)
}
