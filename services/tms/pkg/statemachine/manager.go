package statemachine

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ManagerParams struct {
	fx.In

	Logger *zap.Logger
}

type Manager struct {
	l *zap.Logger

	stopStateMachineFactory     func(stop *shipment.Stop) StateMachine
	moveStateMachineFactory     func(move *shipment.ShipmentMove) StateMachine
	shipmentStateMachineFactory func(shipment *shipment.Shipment) StateMachine
}

func NewManager(p ManagerParams) *Manager {
	manager := &Manager{
		l: p.Logger.With(zap.String("component", "stateMachineManager")),
	}
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

func (m *Manager) CalculateStatuses(shp *shipment.Shipment) error {
	m.l.Debug("calculating statuses", zap.String("shipmentID", shp.ID.String()))

	multiErr := errortypes.NewMultiError()

	shipmentSM := m.ForShipment(shp)

	if shipmentSM.IsInTerminalState() {
		m.l.Debug(
			"shipment in terminal state, skipping status calculation",
			zap.String("shipmentID", shp.ID.String()),
		)
		return nil
	}

	m.processMovesAndStops(shp, multiErr)

	m.processShipmentStatus(shp, shipmentSM, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (m *Manager) CalculateShipmentTimestamps(shp *shipment.Shipment) error {
	m.l.Debug("calculating shipment timestamps", zap.String("shipmentID", shp.ID.String()))

	if len(shp.Moves) == 0 {
		m.l.Debug(
			"no moves found, skipping timestamp calculation",
			zap.String("shipmentID", shp.ID.String()),
		)
		return nil
	}

	firstMove := shp.Moves[0]
	if len(firstMove.Stops) > 0 {
		firstStop := firstMove.Stops[0]
		if firstStop.IsOriginStop() && firstStop.ActualDeparture != nil {
			shp.ActualShipDate = firstStop.ActualDeparture
			m.l.Debug(
				"updated actual ship date",
				zap.String("shipmentID", shp.ID.String()),
				zap.Int64("actualShipDate", *shp.ActualShipDate),
			)
		}
	}

	lastMove := shp.Moves[len(shp.Moves)-1]
	if len(lastMove.Stops) > 0 {
		lastStop := lastMove.Stops[len(lastMove.Stops)-1]
		if lastStop.IsDestinationStop() && lastStop.ActualArrival != nil {
			shp.ActualDeliveryDate = lastStop.ActualArrival
			m.l.Debug(
				"updated actual delivery date",
				zap.String("shipmentID", shp.ID.String()),
				zap.Int64("actualDeliveryDate", *shp.ActualDeliveryDate),
			)
		}
	}

	return nil
}

func (m *Manager) processMovesAndStops(shp *shipment.Shipment, multiErr *errortypes.MultiError) {
	for moveIdx, move := range shp.Moves {
		moveSM := m.ForMove(move)

		if moveSM.IsInTerminalState() {
			continue
		}

		m.processStopsForMove(move, multiErr)

		moveEvent := m.determineMoveEvent(move)
		if moveEvent != nil && moveSM.CanTransition(moveEvent) {
			if err := moveSM.Transition(moveEvent); err != nil {
				multiErr.Add(
					fmt.Sprintf("moves[%d].status", moveIdx),
					errortypes.ErrInvalid,
					err.Error(),
				)
			}
		}
	}
}

type stopTransitionResult struct {
	stopIdx int
	stopID  string
	event   string
	err     error
}

func (m *Manager) processStopsForMove(
	move *shipment.ShipmentMove,
	multiErr *errortypes.MultiError,
) {
	if len(move.Stops) == 0 {
		return
	}

	if len(move.Stops) <= 3 {
		m.processStopsSequentially(move, multiErr)
		return
	}

	m.processStopsInParallel(move, multiErr)
}

func (m *Manager) processStopsSequentially(
	move *shipment.ShipmentMove,
	multiErr *errortypes.MultiError,
) {
	for stopIdx, stop := range move.Stops {
		stopSM := m.ForStop(stop)

		if stopSM.IsInTerminalState() {
			continue
		}

		stopEvent := m.determineStopEvent(stop)
		if stopEvent == nil {
			continue
		}

		if stopSM.CanTransition(stopEvent) {
			m.l.Info(
				"transitioning stop",
				zap.String("stopID", stop.ID.String()),
				zap.String("event", stopEvent.EventType()),
				zap.String("fromState", stopSM.CurrentState()),
			)

			if err := stopSM.Transition(stopEvent); err != nil {
				m.l.Error(
					"failed to transition stop",
					zap.String("stopID", stop.ID.String()),
					zap.String("event", stopEvent.EventType()),
					zap.String("fromState", stopSM.CurrentState()),
					zap.Error(err),
				)

				multiErr.Add(
					fmt.Sprintf("stops[%d].status", stopIdx),
					errortypes.ErrInvalid,
					err.Error(),
				)
			}
		}
	}
}

func (m *Manager) processStopsInParallel(
	move *shipment.ShipmentMove,
	multiErr *errortypes.MultiError,
) {
	stopCount := len(move.Stops)

	p := pool.NewWithResults[*stopTransitionResult]().
		WithMaxGoroutines(min(stopCount, 4)).
		WithContext(context.Background())

	for idx, stop := range move.Stops {
		stopIdx := idx
		currentStop := stop

		p.Go(func(_ context.Context) (*stopTransitionResult, error) {
			stopSM := m.ForStop(currentStop)

			if stopSM.IsInTerminalState() {
				return &stopTransitionResult{}, nil
			}

			stopEvent := m.determineStopEvent(currentStop)
			if stopEvent == nil {
				return &stopTransitionResult{}, nil
			}

			if !stopSM.CanTransition(stopEvent) {
				return &stopTransitionResult{}, nil
			}

			m.l.Info(
				"transitioning stop",
				zap.String("stopID", currentStop.ID.String()),
				zap.String("event", stopEvent.EventType()),
				zap.String("fromState", stopSM.CurrentState()),
			)

			if err := stopSM.Transition(stopEvent); err != nil {
				m.l.Error(
					"failed to transition stop",
					zap.String("stopID", currentStop.ID.String()),
					zap.String("event", stopEvent.EventType()),
					zap.String("fromState", stopSM.CurrentState()),
					zap.Error(err),
				)

				return &stopTransitionResult{
					stopIdx: stopIdx,
					stopID:  currentStop.ID.String(),
					event:   stopEvent.EventType(),
					err:     err,
				}, nil
			}

			return &stopTransitionResult{
				stopIdx: stopIdx,
				stopID:  currentStop.ID.String(),
				event:   stopEvent.EventType(),
			}, nil
		})
	}

	results, _ := p.Wait()

	for _, result := range results {
		// Skip sentinel empty results (no transition occurred)
		if result != nil && result.stopID != "" && result.err != nil {
			multiErr.Add(
				fmt.Sprintf("stops[%d].status", result.stopIdx),
				errortypes.ErrInvalid,
				result.err.Error(),
			)
		}
	}
}

func (m *Manager) determineStopEvent(stop *shipment.Stop) TransitionEvent {
	switch {
	case stop.ActualArrival != nil && stop.ActualDeparture != nil:
		return EventStopDeparted
	case stop.ActualArrival != nil:
		return EventStopArrived
	default:
		return nil
	}
}

func (m *Manager) determineMoveEvent(move *shipment.ShipmentMove) TransitionEvent {
	if len(move.Stops) == 0 {
		return nil
	}

	allStopsCompleted := true
	anyStopInTransit := false
	originCompleted := false

	for i, stop := range move.Stops {
		if !stop.IsCompleted() {
			allStopsCompleted = false
		}
		if stop.IsInTransit() {
			anyStopInTransit = true
		}

		// ! Check if origin stop (first stop) is completed
		if i == 0 && stop.IsOriginStop() && stop.IsCompleted() {
			originCompleted = true
		}
	}

	switch {
	case allStopsCompleted:
		return EventMoveCompleted
	case originCompleted || anyStopInTransit:
		// ! A move is in transit if either:
		// ! 1. The origin stop is completed (vehicle has departed first pickup)
		// ! 2. Any stop is currently in transit
		return EventMoveStarted
	case move.Assignment != nil && move.IsNew():
		// ! Only assign if the move is currently in New status and has an assignment
		return EventMoveAssigned
	default:
		return nil
	}
}

func (m *Manager) processShipmentStatus(
	shp *shipment.Shipment,
	shipmentSM StateMachine,
	multiErr *errortypes.MultiError,
) {
	shipmentEvent := m.determineShipmentEvent(shp)
	if shipmentEvent == nil {
		return
	}

	if shipmentSM.CanTransition(shipmentEvent) {
		if err := shipmentSM.Transition(shipmentEvent); err != nil {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}
	}
}

type aggregateState struct {
	totalMoves     int
	movesCompleted int
	movesInTransit int
	movesAssigned  int
	hasDelayed     bool
}

func (m *Manager) aggregateShipmentState(
	shp *shipment.Shipment,
	currentTime int64,
) *aggregateState {
	state := &aggregateState{
		totalMoves: len(shp.Moves),
	}

	for _, move := range shp.Moves {
		switch {
		case move.IsCompleted():
			state.movesCompleted++
		case move.IsInTransit():
			state.movesInTransit++
		case move.IsAssigned():
			state.movesAssigned++
		}

		if !state.hasDelayed {
			for _, stop := range move.Stops {
				// ! A stop is delayed if it's not completed/canceled and past planned departure
				if !stop.IsCompleted() && !stop.IsCanceled() &&
					stop.PlannedDeparture > 0 &&
					currentTime > stop.PlannedDeparture {
					state.hasDelayed = true
					break // ! Found delay, no need to check more stops in this move
				}
			}
		}
	}

	return state
}

func (m *Manager) determineShipmentEvent(shp *shipment.Shipment) TransitionEvent {
	if len(shp.Moves) == 0 {
		return nil
	}

	currentTime := utils.NowUnix()
	state := m.aggregateShipmentState(shp, currentTime)

	switch {
	// ! Order of these cases is important for precedence
	case state.movesCompleted == state.totalMoves:
		return EventShipmentCompleted
	case state.movesCompleted > 0 && state.movesCompleted < state.totalMoves:
		return EventShipmentPartialCompleted
	case state.movesInTransit > 0 && state.hasDelayed && shp.IsInTransit():
		// ! Delayed event only valid if shipment is ALREADY InTransit
		// ! A shipment in New status cannot go directly to Delayed
		return EventShipmentDelayed
	case state.movesInTransit > 0:
		return EventShipmentInTransit
	case state.movesAssigned == state.totalMoves:
		return EventShipmentAssigned
	case state.movesAssigned > 0 && state.movesAssigned < state.totalMoves:
		return EventShipmentPartiallyAssigned
	default:
		return nil
	}
}
