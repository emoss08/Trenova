package statemachine

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type StateMachineParams struct {
	fx.In

	Logger *logger.Logger
}

type StateMachineManager struct {
	logger *zerolog.Logger

	stopStateMachineFactory     func(stop *shipment.Stop) StateMachine
	moveStateMachineFactory     func(move *shipment.ShipmentMove) StateMachine
	shipmentStateMachineFactory func(shipment *shipment.Shipment) StateMachine
}

type StateMachineManagerParams struct {
	fx.In

	Logger *logger.Logger
}

func NewStateMachineManager(p StateMachineManagerParams) *StateMachineManager {
	log := p.Logger.With().
		Str("component", "state_machine_manager").
		Logger()

	manager := &StateMachineManager{
		logger: &log,
	}

	// Register state machine factories
	manager.stopStateMachineFactory = NewStopStateMachine
	manager.moveStateMachineFactory = NewMoveStateMachine
	manager.shipmentStateMachineFactory = NewShipmentStateMachine

	return manager
}

func (m *StateMachineManager) ForStop(stop *shipment.Stop) StateMachine {
	return m.stopStateMachineFactory(stop)
}

func (m *StateMachineManager) ForMove(move *shipment.ShipmentMove) StateMachine {
	return m.moveStateMachineFactory(move)
}

func (m *StateMachineManager) ForShipment(shp *shipment.Shipment) StateMachine {
	return m.shipmentStateMachineFactory(shp)
}

// CalculateStatuses calculates and updates the statuses of a shipment and all its related entities
func (m *StateMachineManager) CalculateStatuses(ctx context.Context, shp *shipment.Shipment) error {
	m.logger.Debug().
		Str("shipmentID", shp.ID.String()).
		Str("currentStatus", string(shp.Status)).
		Msg("calculating statuses")

	// Multi-error to collect all validation errors
	multiErr := errors.NewMultiError()

	// Get state machines
	shipmentSM := m.ForShipment(shp)

	// Skip processing for terminal states
	if shipmentSM.IsInTerminalState() {
		m.logger.Debug().
			Str("shipmentID", shp.ID.String()).
			Str("status", shipmentSM.CurrentState()).
			Msg("shipment in terminal state, skipping status calculation")
		return nil
	}

	// Process stops and moves first (bottom-up approach)
	for moveIdx, move := range shp.Moves {
		moveSM := m.ForMove(move)

		// Skip if move is in terminal state
		if moveSM.IsInTerminalState() {
			continue
		}

		// Process stop statuses for this move
		for stopIdx, stop := range move.Stops {
			stopSM := m.ForStop(stop)

			// Skip if stop is in terminal state
			if stopSM.IsInTerminalState() {
				continue
			}

			// Determine event for stop based on its data
			var stopEvent TransitionEvent
			switch {
			case stop.ActualArrival != nil && stop.ActualDeparture != nil:
				stopEvent = EventStopDeparted
			case stop.ActualArrival != nil:
				stopEvent = EventStopArrived
			default:
				// No transition needed
				continue
			}

			// Try to transition the stop
			if stopSM.CanTransition(ctx, stopEvent) {
				m.logger.Info().
					Str("stopID", stop.ID.String()).
					Str("event", string(stopEvent)).
					Str("fromState", stopSM.CurrentState()).
					Msg("transitioning stop")

				if err := stopSM.Transition(ctx, stopEvent); err != nil {
					m.logger.Error().
						Str("stopID", stop.ID.String()).
						Str("event", string(stopEvent)).
						Str("fromState", stopSM.CurrentState()).
						Err(err).
						Msg("failed to transition stop")

					multiErr.Add(
						fmt.Sprintf("stops[%d].status", stopIdx),
						errors.ErrInvalid,
						err.Error(),
					)
				}
			}
		}

		// Determine event for move based on stop statuses
		var moveEvent TransitionEvent

		// Check stop states to determine move event
		allStopsCompleted := len(move.Stops) > 0
		anyStopInTransit := false
		originCompleted := false

		for i, stop := range move.Stops {
			if stop.Status != shipment.StopStatusCompleted {
				allStopsCompleted = false
			}
			if stop.Status == shipment.StopStatusInTransit {
				anyStopInTransit = true
			}

			// Check if the origin stop (first pickup) is completed
			if stop.StatusEquals(shipment.StopStatusCompleted) && i == 0 && stop.IsOriginStop() {
				originCompleted = true
			}
		}

		switch {
		case allStopsCompleted:
			moveEvent = EventMoveCompleted
		case originCompleted || anyStopInTransit:
			// * A move is in transit if either:
			// * 1. The origin stop is completed (vehicle has departed first pickup)
			// * 2. Any stop is currently in transit
			moveEvent = EventMoveStarted
		case move.Assignment != nil:
			moveEvent = EventMoveAssigned
		default:
			// No transition needed
			continue
		}
		// Try to transition the move
		if moveSM.CanTransition(ctx, moveEvent) {
			if err := moveSM.Transition(ctx, moveEvent); err != nil {
				multiErr.Add(
					fmt.Sprintf("moves[%d].status", moveIdx),
					errors.ErrInvalid,
					err.Error(),
				)
			}
		}
	}

	// Finally, determine event for shipment based on move statuses
	var shipmentEvent TransitionEvent

	// Analyze move statuses for shipment event
	var (
		totalMoves      = len(shp.Moves)
		movesCompleted  = 0
		movesInTransit  = 0
		movesAssigned   = 0
		hasDelayedMoves = false
	)

	for _, move := range shp.Moves {
		switch move.Status {
		case shipment.MoveStatusCompleted:
			movesCompleted++
		case shipment.MoveStatusInTransit:
			movesInTransit++
		case shipment.MoveStatusAssigned:
			movesAssigned++
		}
	}

	switch {
	case totalMoves == 0:
		// No moves, no state change needed
		return nil
	case movesCompleted == totalMoves:
		shipmentEvent = EventShipmentCompleted
	case movesCompleted > 0 && movesCompleted < totalMoves:
		shipmentEvent = EventShipmentPartialCompleted
	case movesInTransit > 0:
		shipmentEvent = EventShipmentInTransit
	case hasDelayedMoves:
		shipmentEvent = EventShipmentDelayed
	case movesAssigned == totalMoves:
		shipmentEvent = EventShipmentAssigned
	case movesAssigned > 0 && movesAssigned < totalMoves:
		shipmentEvent = EventShipmentPartiallyAssigned
	default:
		// No transition needed
		return nil
	}

	// Try to transition the shipment
	if shipmentSM.CanTransition(ctx, shipmentEvent) {
		if err := shipmentSM.Transition(ctx, shipmentEvent); err != nil {
			multiErr.Add(
				"status",
				errors.ErrInvalid,
				err.Error(),
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
