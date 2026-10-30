package movecompletion

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

const (
	candidateStop = "done_stop"
	candidateMove = "done_move"
)

func At(sm *shipmentdb.ShipmentMovesTable, stp *shipmentdb.StopsTable) keel.Int64Expr {
	return keel.Int64Of(stp.ActualDeparture.Coalesce(stp.ActualArrival).Max()).
		Coalesce(sm.UpdatedAt)
}

func Candidate(sm *shipmentdb.ShipmentMovesTable, start, end int64) keel.Pred {
	stp := shipmentdb.Stops.As(candidateStop)
	done := shipmentdb.ShipmentMoves.As(candidateMove)
	touched := stp.ActualDeparture.Coalesce(stp.ActualArrival)

	return sm.ID.InArray(keel.StringArrayOf(keel.ArrayOf(keel.Union(
		stp.Pick(stp.ShipmentMoveID).Where(touched.Gte(start), touched.Lt(end)),
		done.Pick(done.ID).Where(
			done.Status.Eq(shipmentvocab.MoveStatusCompleted),
			done.UpdatedAt.Gte(start),
			done.UpdatedAt.Lt(end),
		),
	))))
}
