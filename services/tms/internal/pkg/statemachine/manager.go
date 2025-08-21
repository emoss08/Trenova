/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package statemachine

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ManagerParams struct {
	fx.In

	Logger *logger.Logger
}

type Manager struct {
	logger *zerolog.Logger

	stopStateMachineFactory     func(stop *shipment.Stop) StateMachine
	moveStateMachineFactory     func(move *shipment.ShipmentMove) StateMachine
	shipmentStateMachineFactory func(shipment *shipment.Shipment) StateMachine
}

func NewManager(p ManagerParams) *Manager {
	log := p.Logger.With().
		Str("component", "state_machine_manager").
		Logger()

	manager := &Manager{
		logger: &log,
	}

	// Register state machine factories
	manager.stopStateMachineFactory = NewStopStateMachine
	manager.moveStateMachineFactory = NewMoveStateMachine
	manager.shipmentStateMachineFactory = NewShipmentStateMachine

	return manager
}

func (m *Manager) ForStop(stop *shipment.Stop) StateMachine {
	return m.stopStateMachineFactory(stop)
}

func (m *Manager) ForMove(move *shipment.ShipmentMove) StateMachine {
	return m.moveStateMachineFactory(move)
}

func (m *Manager) ForShipment(shp *shipment.Shipment) StateMachine {
	return m.shipmentStateMachineFactory(shp)
}

// CalculateStatuses calculates and updates the statuses of a shipment and all its related entities
func (m *Manager) CalculateStatuses(shp *shipment.Shipment) error {
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
	m.processMovesAndStops(shp, multiErr)

	// Process shipment status based on move statuses
	m.processShipmentStatus(shp, shipmentSM, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (m *Manager) CalculateShipmentTimestamps(shp *shipment.Shipment) error {
	m.logger.Debug().
		Str("shipmentID", shp.ID.String()).
		Msg("calculating shipment timestamps")

	if len(shp.Moves) == 0 {
		m.logger.Debug().
			Str("shipmentID", shp.ID.String()).
			Msg("no moves found, skipping timestamp calculation")
		return nil
	}

	// Calculate ActualShipDate
	// This is the ActualDeparture of the first stop of the first move
	firstMove := shp.Moves[0]
	if len(firstMove.Stops) > 0 {
		firstStop := firstMove.Stops[0]
		// Ensure it's the origin stop and has departed
		if firstStop.IsOriginStop() && firstStop.ActualDeparture != nil {
			shp.ActualShipDate = firstStop.ActualDeparture
			m.logger.Debug().
				Str("shipmentID", shp.ID.String()).
				Int64("actualShipDate", *shp.ActualShipDate).
				Msg("updated actual ship date")
		}
	}

	// Calculate ActualDeliveryDate
	// This is the ActualArrival of the last stop of the last move
	lastMove := shp.Moves[len(shp.Moves)-1]
	if len(lastMove.Stops) > 0 {
		lastStop := lastMove.Stops[len(lastMove.Stops)-1]
		// Ensure it's the destination stop and has arrived
		if lastStop.IsDestinationStop() && lastStop.ActualArrival != nil {
			shp.ActualDeliveryDate = lastStop.ActualArrival
			m.logger.Debug().
				Str("shipmentID", shp.ID.String()).
				Int64("actualDeliveryDate", *shp.ActualDeliveryDate).
				Msg("updated actual delivery date")
		}
	}

	return nil
}

// processMovesAndStops processes the status transitions for moves and their stops
func (m *Manager) processMovesAndStops(shp *shipment.Shipment, multiErr *errors.MultiError) {
	for moveIdx, move := range shp.Moves {
		moveSM := m.ForMove(move)

		// Skip if move is in terminal state
		if moveSM.IsInTerminalState() {
			continue
		}

		// Process stops for this move
		m.processStopsForMove(move, multiErr)

		// Determine and apply move status transition
		moveEvent := m.determineMoveEvent(move)
		if moveEvent != nil && moveSM.CanTransition(moveEvent) {
			if err := moveSM.Transition(moveEvent); err != nil {
				multiErr.Add(
					fmt.Sprintf("moves[%d].status", moveIdx),
					errors.ErrInvalid,
					err.Error(),
				)
			}
		}
	}
}

// processStopsForMove processes status transitions for stops in a move
func (m *Manager) processStopsForMove(move *shipment.ShipmentMove, multiErr *errors.MultiError) {
	for stopIdx, stop := range move.Stops {
		stopSM := m.ForStop(stop)

		// Skip if stop is in terminal state
		if stopSM.IsInTerminalState() {
			continue
		}

		// Determine event for stop based on its data
		stopEvent := m.determineStopEvent(stop)
		if stopEvent == nil {
			continue // No transition needed
		}

		// Try to transition the stop
		if stopSM.CanTransition(stopEvent) {
			m.logger.Info().
				Str("stopID", stop.ID.String()).
				Str("event", stopEvent.EventType()).
				Str("fromState", stopSM.CurrentState()).
				Msg("transitioning stop")

			if err := stopSM.Transition(stopEvent); err != nil {
				m.logger.Error().
					Str("stopID", stop.ID.String()).
					Str("event", stopEvent.EventType()).
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
}

// determineStopEvent determines the appropriate event for a stop based on its data
func (m *Manager) determineStopEvent(stop *shipment.Stop) TransitionEvent {
	switch {
	case stop.ActualArrival != nil && stop.ActualDeparture != nil:
		return EventStopDeparted
	case stop.ActualArrival != nil:
		return EventStopArrived
	default:
		return nil // No transition needed
	}
}

// determineMoveEvent determines the appropriate event for a move based on its stops
func (m *Manager) determineMoveEvent(move *shipment.ShipmentMove) TransitionEvent {
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
		return EventMoveCompleted
	case originCompleted || anyStopInTransit:
		// A move is in transit if either:
		// 1. The origin stop is completed (vehicle has departed first pickup)
		// 2. Any stop is currently in transit
		return EventMoveStarted
	case move.Assignment != nil && move.Status == shipment.MoveStatusNew:
		// Only assign if the move is currently in New status and has an assignment
		return EventMoveAssigned
	default:
		return nil // No transition needed
	}
}

// processShipmentStatus processes the status transition for a shipment based on its moves
func (m *Manager) processShipmentStatus(
	shp *shipment.Shipment,
	shipmentSM StateMachine,
	multiErr *errors.MultiError,
) {
	shipmentEvent := m.determineShipmentEvent(shp)
	if shipmentEvent == nil {
		return // No transition needed
	}

	// Try to transition the shipment
	if shipmentSM.CanTransition(shipmentEvent) {
		if err := shipmentSM.Transition(shipmentEvent); err != nil {
			multiErr.Add(
				"status",
				errors.ErrInvalid,
				err.Error(),
			)
		}
	}
}

// hasDelayedStops checks if any active stop in the shipment is past its planned arrival time.
func (m *Manager) hasDelayedStops(shp *shipment.Shipment, currentTime int64) bool {
	for _, move := range shp.Moves {
		for _, stop := range move.Stops {
			// ! A stop contributes to delay if it's not completed or canceled,
			// ! has a valid planned arrival time, and that time is in the past.
			if stop.Status != shipment.StopStatusCompleted &&
				stop.Status != shipment.StopStatusCanceled &&
				stop.PlannedArrival > 0 && // ! 0 is not a valid/set planned time
				currentTime > stop.PlannedArrival {

				return true
			}
		}
	}

	return false
}

// determineShipmentEvent determines the appropriate event for a shipment based on its moves
func (m *Manager) determineShipmentEvent(shp *shipment.Shipment) TransitionEvent {
	var (
		totalMoves     = len(shp.Moves)
		movesCompleted = 0
		movesInTransit = 0
		movesAssigned  = 0
	)

	if totalMoves == 0 { // Early exit if there are no moves
		return nil
	}

	currentTime := timeutils.NowUnix()
	hasDelayedMoves := m.hasDelayedStops(shp, currentTime)

	for _, move := range shp.Moves {
		//nolint:exhaustive // No need to include the terminal states
		switch move.Status {
		case shipment.MoveStatusCompleted:
			movesCompleted++
		case shipment.MoveStatusInTransit:
			movesInTransit++
		case shipment.MoveStatusAssigned:
			movesAssigned++
			// New and Canceled moves don't affect these counters for primary event determination
		}
	}

	switch {
	// Order of these cases is important for precedence
	case movesCompleted == totalMoves:
		return EventShipmentCompleted
	case movesCompleted > 0 && movesCompleted < totalMoves:
		return EventShipmentPartialCompleted
	case movesInTransit > 0:
		return EventShipmentInTransit
	case hasDelayedMoves && movesInTransit > 0:
		// Only consider delays if shipment has moves that are actually in transit
		return EventShipmentDelayed
	case movesAssigned == totalMoves:
		return EventShipmentAssigned
	case movesAssigned > 0 && movesAssigned < totalMoves:
		return EventShipmentPartiallyAssigned
	default:
		return nil // No specific event derived from move statuses
	}
}
