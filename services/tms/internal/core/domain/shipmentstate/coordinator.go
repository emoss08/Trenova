//nolint:cyclop // existing legacy workflow/API shape is intentionally kept stable
package shipmentstate

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Coordinator struct {
	now func() int64
}

func NewCoordinator() *Coordinator {
	return &Coordinator{
		now: timeutils.NowUnix,
	}
}

func NewCoordinatorWithClock(now func() int64) *Coordinator {
	return &Coordinator{now: now}
}

func (c *Coordinator) PrepareForCreate(entity *shipment.Shipment) *errortypes.MultiError {
	return c.PrepareForCreateWithDelayThreshold(entity, 0)
}

func (c *Coordinator) PrepareForCreateWithDelayThreshold(
	entity *shipment.Shipment,
	delayThresholdMinutes int16,
) *errortypes.MultiError {
	return c.prepare(nil, entity, delayThresholdMinutes)
}

func (c *Coordinator) PrepareForUpdate(original, entity *shipment.Shipment) *errortypes.MultiError {
	return c.PrepareForUpdateWithDelayThreshold(original, entity, 0)
}

func (c *Coordinator) PrepareForUpdateWithDelayThreshold(
	original, entity *shipment.Shipment,
	delayThresholdMinutes int16,
) *errortypes.MultiError {
	return c.prepare(original, entity, delayThresholdMinutes)
}

func (c *Coordinator) RefreshShipmentState(entity *shipment.Shipment) {
	c.RefreshShipmentStateWithDelayThreshold(entity, 0)
}

func (c *Coordinator) RefreshShipmentStateWithDelayThreshold(
	entity *shipment.Shipment,
	delayThresholdMinutes int16,
) {
	c.calculateShipmentTimestamps(entity)
	if preservesShipmentStatusOnRefresh(entity.Status) {
		return
	}

	entity.Status = deriveShipmentStatus(entity, c.now(), delayThresholdMinutes)
}

func (c *Coordinator) prepare( //nolint:gocognit // legacy workflow
	original, entity *shipment.Shipment,
	delayThresholdMinutes int16,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove)
	originalStops := make(map[pulid.ID]map[pulid.ID]*shipment.Stop)

	if original != nil {
		for _, move := range original.Moves {
			if move == nil || move.ID.IsNil() {
				continue
			}

			originalMoves[move.ID] = move

			if len(move.Stops) == 0 {
				continue
			}

			stops := make(map[pulid.ID]*shipment.Stop, len(move.Stops))
			for _, stop := range move.Stops {
				if stop == nil || stop.ID.IsNil() {
					continue
				}
				stops[stop.ID] = stop
			}

			originalStops[move.ID] = stops
		}
	}

	for moveIndex, move := range entity.Moves {
		if move == nil {
			continue
		}

		var originalMove *shipment.ShipmentMove
		if !move.ID.IsNil() {
			originalMove = originalMoves[move.ID]
		}
		if move.Assignment == nil &&
			move.Status != shipment.MoveStatusNew &&
			originalMove != nil &&
			originalMove.Assignment != nil {
			move.Assignment = originalMove.Assignment
		}

		moveStops := originalStops[move.ID]
		for stopIndex, stop := range move.Stops {
			if stop == nil {
				continue
			}

			var originalStop *shipment.Stop
			if moveStops != nil && !stop.ID.IsNil() {
				originalStop = moveStops[stop.ID]
			}

			stopPath := fmt.Sprintf("moves[%d].stops[%d]", moveIndex, stopIndex)
			stop.Status = c.resolveStopStatus(
				originalStop,
				stop,
				stopPath,
				multiErr,
			)
		}

		move.Status = c.resolveMoveStatus(
			originalMove,
			move,
			fmt.Sprintf("moves[%d].status", moveIndex),
			multiErr,
		)
	}

	c.calculateShipmentTimestamps(entity)
	entity.Status = c.resolveShipmentStatus(original, entity, delayThresholdMinutes, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (c *Coordinator) resolveStopStatus(
	original *shipment.Stop,
	stop *shipment.Stop,
	path string,
	multiErr *errortypes.MultiError,
) shipment.StopStatus {
	requested := stop.Status
	current := defaultStopStatus(original)
	statusField := path + ".status"

	if original != nil && original.IsCanceled() && requested != shipment.StopStatusCanceled {
		multiErr.Add(
			statusField,
			errortypes.ErrInvalidOperation,
			"Canceled stop cannot transition to another status",
		)
		return original.Status
	}

	if requested == shipment.StopStatusCanceled {
		if !isAllowedStopStatusTransition(current, shipment.StopStatusCanceled) {
			multiErr.Add(
				statusField,
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"Stop status transition from %s to %s is not allowed",
					current,
					requested,
				),
			)
			return current
		}

		return shipment.StopStatusCanceled
	}

	if original != nil && original.IsCanceled() {
		return original.Status
	}

	if original != nil && original.IsCompleted() {
		if !c.validateCompletedStopActuals(stop, path, multiErr) {
			return original.Status
		}

		return shipment.StopStatusCompleted
	}

	switch {
	case stop.ActualArrival != nil && stop.ActualDeparture != nil:
		return shipment.StopStatusCompleted
	case stop.ActualArrival != nil || stop.ActualDeparture != nil:
		return shipment.StopStatusInTransit
	default:
		return shipment.StopStatusNew
	}
}

func (c *Coordinator) resolveMoveStatus(
	original *shipment.ShipmentMove,
	move *shipment.ShipmentMove,
	field string,
	multiErr *errortypes.MultiError,
) shipment.MoveStatus {
	requested := move.Status
	current := defaultMoveStatus(original)

	if original != nil && original.IsCanceled() && requested != shipment.MoveStatusCanceled {
		multiErr.Add(
			field,
			errortypes.ErrInvalidOperation,
			"Canceled move cannot transition to another status",
		)
		return original.Status
	}

	if requested == shipment.MoveStatusCanceled {
		if !isAllowedMoveStatusTransition(current, shipment.MoveStatusCanceled) {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"Move status transition from %s to %s is not allowed",
					current,
					requested,
				),
			)
			return current
		}

		return shipment.MoveStatusCanceled
	}

	if original != nil && original.IsCanceled() {
		return original.Status
	}

	derived := deriveMoveStatus(move)
	if original != nil && original.IsCompleted() {
		if derived != shipment.MoveStatusCompleted {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				"Completed move cannot be reopened by a shipment update",
			)
		}

		return shipment.MoveStatusCompleted
	}

	if derived == shipment.MoveStatusAssigned &&
		!isAllowedMoveStatusTransition(current, shipment.MoveStatusAssigned) {
		multiErr.Add(
			field,
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("Move status transition from %s to %s is not allowed", current, derived),
		)
		return current
	}

	if derived == shipment.MoveStatusNew &&
		current == shipment.MoveStatusAssigned &&
		requested == shipment.MoveStatusNew &&
		move.Assignment == nil {
		return shipment.MoveStatusNew
	}

	if derived == shipment.MoveStatusNew && current == shipment.MoveStatusAssigned {
		return shipment.MoveStatusAssigned
	}

	if derived == shipment.MoveStatusNew && requested == shipment.MoveStatusAssigned {
		if !isAllowedMoveStatusTransition(current, shipment.MoveStatusAssigned) {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"Move status transition from %s to %s is not allowed",
					current,
					requested,
				),
			)
			return current
		}

		return shipment.MoveStatusAssigned
	}

	return derived
}

func (c *Coordinator) resolveShipmentStatus(
	original *shipment.Shipment,
	entity *shipment.Shipment,
	delayThresholdMinutes int16,
	multiErr *errortypes.MultiError,
) shipment.Status {
	requested := entity.Status
	current := defaultShipmentStatus(original)
	derived := deriveShipmentStatus(entity, c.now(), delayThresholdMinutes)

	if original != nil && original.StatusEquals(shipment.StatusCanceled) &&
		requested != shipment.StatusCanceled {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"Canceled shipment cannot transition to another status",
		)
		return original.Status
	}

	if original != nil && original.StatusEquals(shipment.StatusInvoiced) &&
		requested != shipment.StatusInvoiced {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"Invoiced shipment cannot transition to another status",
		)
		return original.Status
	}

	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch requested {
	case shipment.StatusCanceled:
		if !isAllowedShipmentStatusTransition(current, shipment.StatusCanceled) {
			multiErr.Add(
				"status",
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"Shipment status transition from %s to %s is not allowed",
					current,
					requested,
				),
			)
			return derived
		}

		return shipment.StatusCanceled
	case shipment.StatusReadyToInvoice, shipment.StatusInvoiced:
		if derived != shipment.StatusCompleted &&
			(original == nil || !allowsBillingContinuation(original.Status)) {
			multiErr.Add(
				"status",
				errortypes.ErrInvalidOperation,
				fmt.Sprintf("Shipment cannot transition to %s until it is completed", requested),
			)
			return derived
		}

		if !isAllowedShipmentStatusTransition(current, requested) {
			multiErr.Add(
				"status",
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"Shipment status transition from %s to %s is not allowed",
					current,
					requested,
				),
			)
			return derived
		}

		return requested
	default:
		return derived
	}
}

func (c *Coordinator) calculateShipmentTimestamps(entity *shipment.Shipment) {
	entity.ActualShipDate = nil
	entity.ActualDeliveryDate = nil

	if len(entity.Moves) == 0 {
		return
	}

	firstMove := firstActiveMoveBySequence(entity.Moves)
	firstStop := firstActiveStopBySequence(firstMove)
	if firstStop != nil && firstStop.IsOriginStop() && firstStop.ActualDeparture != nil {
		entity.ActualShipDate = firstStop.ActualDeparture
	}

	lastMove := lastActiveMoveBySequence(entity.Moves)
	lastStop := lastActiveStopBySequence(lastMove)
	if lastStop != nil && lastStop.IsDestinationStop() && lastStop.ActualArrival != nil {
		entity.ActualDeliveryDate = lastStop.ActualArrival
	}
}

func (c *Coordinator) validateCompletedStopActuals(
	stop *shipment.Stop,
	path string,
	multiErr *errortypes.MultiError,
) bool {
	valid := true
	if stop.ActualArrival == nil {
		multiErr.Add(
			path+".actualArrival",
			errortypes.ErrRequired,
			"Completed stop must keep an actual arrival",
		)
		valid = false
	}

	if stop.ActualDeparture == nil {
		multiErr.Add(
			path+".actualDeparture",
			errortypes.ErrRequired,
			"Completed stop must keep an actual departure",
		)
		valid = false
	}

	if !valid {
		return false
	}

	now := c.now()
	if *stop.ActualArrival > now {
		multiErr.Add(
			path+".actualArrival",
			errortypes.ErrInvalid,
			"Actual arrival cannot be in the future",
		)
		valid = false
	}

	if *stop.ActualDeparture > now {
		multiErr.Add(
			path+".actualDeparture",
			errortypes.ErrInvalid,
			"Actual departure cannot be in the future",
		)
		valid = false
	}

	if *stop.ActualDeparture < *stop.ActualArrival {
		multiErr.Add(
			path+".actualDeparture",
			errortypes.ErrInvalid,
			"Actual departure must be greater than or equal to actual arrival",
		)
		valid = false
	}

	return valid
}

func firstActiveMoveBySequence(moves []*shipment.ShipmentMove) *shipment.ShipmentMove {
	var candidate *shipment.ShipmentMove
	for _, move := range moves {
		if move == nil || move.IsCanceled() {
			continue
		}

		if candidate == nil || move.Sequence < candidate.Sequence {
			candidate = move
		}
	}

	return candidate
}

func lastActiveMoveBySequence(moves []*shipment.ShipmentMove) *shipment.ShipmentMove {
	var candidate *shipment.ShipmentMove
	for _, move := range moves {
		if move == nil || move.IsCanceled() {
			continue
		}

		if candidate == nil || move.Sequence > candidate.Sequence {
			candidate = move
		}
	}

	return candidate
}

func firstActiveStopBySequence(move *shipment.ShipmentMove) *shipment.Stop {
	if move == nil {
		return nil
	}

	var candidate *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || stop.IsCanceled() {
			continue
		}

		if candidate == nil || stop.Sequence < candidate.Sequence {
			candidate = stop
		}
	}

	return candidate
}

func lastActiveStopBySequence(move *shipment.ShipmentMove) *shipment.Stop {
	if move == nil {
		return nil
	}

	var candidate *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || stop.IsCanceled() {
			continue
		}

		if candidate == nil || stop.Sequence > candidate.Sequence {
			candidate = stop
		}
	}

	return candidate
}

func deriveMoveStatus(move *shipment.ShipmentMove) shipment.MoveStatus {
	allCompleted := true
	anyInTransit := false
	originCompleted := false
	hasActiveStop := false

	for stopIndex, stop := range move.Stops {
		if stop == nil || stop.IsCanceled() {
			continue
		}

		hasActiveStop = true

		if !stop.IsCompleted() {
			allCompleted = false
		}

		if stop.IsInTransit() {
			anyInTransit = true
		}

		if stopIndex == 0 && stop.IsOriginStop() && stop.IsCompleted() {
			originCompleted = true
		}
	}

	switch {
	case hasActiveStop && allCompleted:
		return shipment.MoveStatusCompleted
	case originCompleted || anyInTransit:
		return shipment.MoveStatusInTransit
	case move != nil && move.HasAssignment():
		return shipment.MoveStatusAssigned
	default:
		return shipment.MoveStatusNew
	}
}

func deriveShipmentStatus(
	entity *shipment.Shipment,
	currentTime int64,
	delayThresholdMinutes int16,
) shipment.Status {
	if len(entity.Moves) == 0 {
		return shipment.StatusNew
	}

	totalMoves := 0
	completed := 0
	inTransit := 0
	assigned := 0
	delayed := false

	for _, move := range entity.Moves {
		if move == nil || move.IsCanceled() {
			continue
		}

		totalMoves++

		switch {
		case move.IsCompleted():
			completed++
		case move.IsInTransit():
			inTransit++
		case move.IsAssigned():
			assigned++
		}

		if delayed {
			continue
		}

		for _, stop := range move.Stops {
			if IsStopOverdue(stop, currentTime, delayThresholdMinutes) {
				delayed = true
				break
			}
		}
	}

	if totalMoves == 0 {
		return shipment.StatusNew
	}

	switch {
	case completed == totalMoves:
		return shipment.StatusCompleted
	case delayed:
		return shipment.StatusDelayed
	case completed > 0:
		return shipment.StatusPartiallyCompleted
	case inTransit > 0:
		return shipment.StatusInTransit
	case assigned == totalMoves:
		return shipment.StatusAssigned
	case assigned > 0:
		return shipment.StatusPartiallyAssigned
	default:
		return shipment.StatusNew
	}
}

func defaultStopStatus(original *shipment.Stop) shipment.StopStatus {
	if original != nil && original.Status != "" {
		return original.Status
	}

	return shipment.StopStatusNew
}

func defaultMoveStatus(original *shipment.ShipmentMove) shipment.MoveStatus {
	if original != nil && original.Status != "" {
		return original.Status
	}

	return shipment.MoveStatusNew
}

func defaultShipmentStatus(original *shipment.Shipment) shipment.Status {
	if original != nil && original.Status != "" {
		return original.Status
	}

	return shipment.StatusNew
}

func allowsBillingContinuation(status shipment.Status) bool {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch status {
	case shipment.StatusReadyToInvoice, shipment.StatusInvoiced:
		return true
	default:
		return false
	}
}

func preservesShipmentStatusOnRefresh(status shipment.Status) bool {
	//nolint:exhaustive // only billing and terminal statuses are preserved during refresh
	switch status {
	case shipment.StatusCanceled,
		shipment.StatusReadyToInvoice,
		shipment.StatusCompleted,
		shipment.StatusInvoiced:
		return true
	default:
		return false
	}
}
