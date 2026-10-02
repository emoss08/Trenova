package shipmentstate

import "github.com/emoss08/trenova/internal/core/domain/shipment"

func (c *Coordinator) PrepareForUncancel(entity *shipment.Shipment, delayThresholdMinutes int16) {
	if entity == nil {
		return
	}

	entity.ApplyUncancel()

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		for _, stop := range move.Stops {
			if stop != nil && stop.IsCanceled() {
				stop.Status = stopStatusFromActuals(stop)
			}
		}

		if move.Assignment != nil && move.Assignment.Status == shipment.AssignmentStatusCanceled {
			move.Assignment.Status = shipment.AssignmentStatusNew
		}

		if move.IsCanceled() {
			move.Status = deriveMoveStatus(move)
		}
	}

	entity.Status = deriveShipmentStatus(entity, c.now(), delayThresholdMinutes)
}

func CancelPreservedStopStatuses() []shipment.StopStatus {
	return []shipment.StopStatus{shipment.StopStatusCompleted, shipment.StopStatusCanceled}
}

func CancelPreservedMoveStatuses() []shipment.MoveStatus {
	return []shipment.MoveStatus{shipment.MoveStatusCompleted, shipment.MoveStatusCanceled}
}

func CancelPreservedAssignmentStatuses() []shipment.AssignmentStatus {
	return []shipment.AssignmentStatus{
		shipment.AssignmentStatusCompleted,
		shipment.AssignmentStatusCanceled,
	}
}
