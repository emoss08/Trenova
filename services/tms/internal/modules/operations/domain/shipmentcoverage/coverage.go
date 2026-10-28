package shipmentcoverage

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

// CloneForUpdate deep-copies the mutable graph the coverage flows touch —
// moves, their driver and carrier assignments, and stops — so a service can
// derive an updated shipment without mutating the loaded original.
func CloneForUpdate(source *shipment.Shipment) *shipment.Shipment {
	if source == nil {
		return nil
	}

	clone := *source
	clone.Moves = make([]*shipment.ShipmentMove, 0, len(source.Moves))

	for _, move := range source.Moves {
		if move == nil {
			clone.Moves = append(clone.Moves, nil)
			continue
		}

		moveClone := *move
		if move.Assignment != nil {
			assignmentClone := *move.Assignment
			moveClone.Assignment = &assignmentClone
		}
		if move.CarrierAssignment != nil {
			carrierAssignmentClone := *move.CarrierAssignment
			moveClone.CarrierAssignment = &carrierAssignmentClone
		}
		moveClone.Stops = make([]*shipment.Stop, 0, len(move.Stops))

		for _, stop := range move.Stops {
			if stop == nil {
				moveClone.Stops = append(moveClone.Stops, nil)
				continue
			}

			stopClone := *stop
			moveClone.Stops = append(moveClone.Stops, &stopClone)
		}

		clone.Moves = append(clone.Moves, &moveClone)
	}

	return &clone
}
