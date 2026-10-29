package shipmentstate

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

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

		if move.Assignment != nil &&
			move.Assignment.Status == shipmentvocab.AssignmentStatusCanceled {
			move.Assignment.Status = shipmentvocab.AssignmentStatusNew
		}

		if move.IsCanceled() {
			move.Status = deriveMoveStatus(move)
		}
	}

	entity.Status = deriveShipmentStatus(entity, c.now(), delayThresholdMinutes)
}

func CancelPreservedStopStatuses() []shipmentvocab.StopStatus {
	return []shipmentvocab.StopStatus{
		shipmentvocab.StopStatusCompleted,
		shipmentvocab.StopStatusCanceled,
	}
}

func CancelPreservedMoveStatuses() []shipmentvocab.MoveStatus {
	return []shipmentvocab.MoveStatus{
		shipmentvocab.MoveStatusCompleted,
		shipmentvocab.MoveStatusCanceled,
	}
}

func CancelPreservedAssignmentStatuses() []shipmentvocab.AssignmentStatus {
	return []shipmentvocab.AssignmentStatus{
		shipmentvocab.AssignmentStatusCompleted,
		shipmentvocab.AssignmentStatusCanceled,
	}
}
