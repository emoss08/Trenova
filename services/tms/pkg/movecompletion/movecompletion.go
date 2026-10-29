package movecompletion

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

const (
	candidatesCTE = "completion_candidates"
	candidateID   = "id"
	candidateStop = "done_stop"
	candidateMove = "done_move"
)

func At(sm *shipmentdb.ShipmentMovesTable, stp *shipmentdb.StopsTable) keel.Int64Expr {
	return keel.Int64Of(stp.ActualDeparture.Coalesce(stp.ActualArrival).Max()).
		Coalesce(sm.UpdatedAt)
}

func Candidates(start, end int64) *keel.CTETable {
	stp := shipmentdb.Stops.As(candidateStop)
	sm := shipmentdb.ShipmentMoves.As(candidateMove)
	touched := stp.ActualDeparture.Coalesce(stp.ActualArrival)

	return keel.CTE(candidatesCTE, []string{candidateID}, keel.Union(
		stp.Pick(stp.ShipmentMoveID).Where(touched.Gte(start), touched.Lt(end)),
		sm.Pick(sm.ID).Where(
			sm.Status.Eq(shipmentvocab.MoveStatusCompleted),
			sm.UpdatedAt.Gte(start),
			sm.UpdatedAt.Lt(end),
		),
	))
}

func ID(candidates *keel.CTETable) keel.AnyCol {
	return candidates.Col(candidateID)
}
