package statemachine

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Manager struct {
	logger *zerolog.Logger

	stopStateMachineFactory     func(stop *shipment.Stop) StateMachine
	moveStateMachineFactory     func(move *shipment.ShipmentMove) StateMachine
	shipmentStateMachineFactory func(shipment *shipment.Shipment) StateMachine
}

type ManagerParams struct {
	fx.In

	Logger *logger.Logger
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
	case move.Assignment != nil:
		return EventMoveAssigned
	default:
		return nil // No transition needed
	}
}

// processShipmentStatus processes the status transition for a shipment based on its moves
func (m *Manager) processShipmentStatus(shp *shipment.Shipment, shipmentSM StateMachine, multiErr *errors.MultiError) {
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

// determineShipmentEvent determines the appropriate event for a shipment based on its moves
func (m *Manager) determineShipmentEvent(shp *shipment.Shipment) TransitionEvent {
	// * Analyze move statuses for shipment event
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
		case shipment.MoveStatusNew:
			// * New moves don't affect any counters
		case shipment.MoveStatusCanceled:
			// * Canceled moves don't affect any counters
		}
	}

	switch {
	case totalMoves == 0:
		return nil // * No moves, no state change needed
	case movesCompleted == totalMoves:
		return EventShipmentCompleted
	case movesCompleted > 0 && movesCompleted < totalMoves:
		return EventShipmentPartialCompleted
	case movesInTransit > 0:
		return EventShipmentInTransit
	case hasDelayedMoves:
		return EventShipmentDelayed
	case movesAssigned == totalMoves:
		return EventShipmentAssigned
	case movesAssigned > 0 && movesAssigned < totalMoves:
		return EventShipmentPartiallyAssigned
	default:
		return nil // * No transition needed
	}
}
