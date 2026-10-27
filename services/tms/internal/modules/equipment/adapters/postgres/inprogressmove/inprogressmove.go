package inprogressmove

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/pkg/keelkinds"
)

type Fields struct {
	MoveID    keel.Expr
	ProNumber keel.Expr
	DueAt     keel.Expr
}

func MoveID(
	moves equipmentcontinuityports.MoveColumns,
	equipment keelkinds.PulidIDCol,
	equipmentID keel.Expr,
) keel.Expr {
	return keel.Scalar(
		keel.Handle[*keel.Row]{Ref: moves.AssignmentMoveID.Ref}.
			Pick(moves.AssignmentMoveID).
			Join(
				keel.Handle[*keel.Row]{Ref: moves.MoveID.Ref},
				moves.MoveID.EqCol(moves.AssignmentMoveID),
			).
			Where(
				equipment.EqCol(equipmentID),
				moves.AssignmentArchivedAt.IsNull(),
				moves.MoveInProgress,
			).
			Order(moves.AssignmentMoveID.Asc()).
			Limit(1),
	)
}

func Of(
	moves equipmentcontinuityports.MoveColumns,
	equipment keelkinds.PulidIDCol,
	equipmentID keel.Expr,
) Fields {
	moveID := MoveID(moves, equipment, equipmentID)

	return Fields{
		MoveID: moveID,
		ProNumber: keel.Scalar(
			keel.Handle[*keel.Row]{Ref: moves.ShipmentID.Ref}.
				Pick(moves.ShipmentProNumber).
				Join(
					keel.Handle[*keel.Row]{Ref: moves.MoveID.Ref},
					moves.MoveShipmentID.EqCol(moves.ShipmentID),
				).
				Where(moves.MoveID.EqCol(moveID)).
				Limit(1),
		),
		DueAt: keel.Scalar(
			keel.Handle[*keel.Row]{Ref: moves.StopMoveID.Ref}.
				Pick(moves.StopWindowEnd.Coalesce(moves.StopWindowStart)).
				Where(moves.StopMoveID.EqCol(moveID)).
				Order(moves.StopSequence.Desc()).
				Limit(1),
		),
	}
}
