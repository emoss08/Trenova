package shipment

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

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
	case shipmentvocab.MoveStatusCompleted:
		return errortypes.NewBusinessError("Completed shipment moves cannot be assigned")
	case shipmentvocab.MoveStatusCanceled:
		return errortypes.NewBusinessError("Canceled shipment moves cannot be assigned")
	default:
		return nil
	}
}
