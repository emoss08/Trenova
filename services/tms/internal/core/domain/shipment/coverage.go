package shipment

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// CloneForUpdate deep-copies the mutable graph the coverage flows touch —
// moves, their driver and carrier assignments, and stops — so a service can
// derive an updated shipment without mutating the loaded original.
func CloneForUpdate(source *Shipment) *Shipment {
	if source == nil {
		return nil
	}

	clone := *source
	clone.Moves = make([]*ShipmentMove, 0, len(source.Moves))

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
		moveClone.Stops = make([]*Stop, 0, len(move.Stops))

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

// FindMove returns the shipment's move with the given ID, or nil when the
// shipment does not contain it.
func (s *Shipment) FindMove(moveID pulid.ID) *ShipmentMove {
	for _, move := range s.Moves {
		if move != nil && move.ID == moveID {
			return move
		}
	}

	return nil
}

// EnsureAssignable rejects coverage changes on moves in a terminal state.
func (m *ShipmentMove) EnsureAssignable() error {
	//nolint:exhaustive // only terminal enum states require explicit handling here
	switch m.Status {
	case MoveStatusCompleted:
		return errortypes.NewBusinessError("Completed shipment moves cannot be assigned")
	case MoveStatusCanceled:
		return errortypes.NewBusinessError("Canceled shipment moves cannot be assigned")
	default:
		return nil
	}
}
