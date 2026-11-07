package driverstart

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
)

const PositionFreshnessSeconds = int64(4 * 3600)

var OpenMoveStatuses = []shipmentvocab.MoveStatus{
	shipmentvocab.MoveStatusAssigned,
	shipmentvocab.MoveStatusInTransit,
}

const (
	finalDropWorker    = "worker_id"
	finalDropLatitude  = "latitude"
	finalDropLongitude = "longitude"
	finalDropEndsAt    = "ends_at"
)

type Start struct {
	Latitude  keel.Float64Expr
	Longitude keel.Float64Expr
	FreeAt    keel.Int64Expr
}

func Join(
	q keel.Query[*worker.Worker],
	w *workerdb.WorkersTable,
	wt *tractordb.TractorsTable,
	now int64,
) (keel.Query[*worker.Worker], Start) {
	job := finalDrops()
	pos := telematicsdb.VehiclePositions.As("ds_pos")
	ec := equipmentcontinuitydb.EquipmentContinuities.As("ds_drop")
	drop := locationdb.Locations.As("ds_drop_loc")

	q = q.
		With(job).
		LeftJoin(job, job.Col(finalDropWorker).EqCol(w.ID)).
		LeftJoin(pos,
			pos.TractorID.EqCol(wt.ID),
			pos.RecordedAt.Gte(now-PositionFreshnessSeconds),
		).
		LeftJoin(ec,
			ec.EquipmentID.EqCol(wt.ID),
			ec.EquipmentType.Eq(equipmentcontinuity.EquipmentTypeTractor),
			ec.IsCurrent.IsTrue(),
		).
		LeftJoin(drop, drop.ID.EqCol(ec.CurrentLocationID))

	return q, Start{
		Latitude: keel.Float64Of(job.Col(finalDropLatitude)).
			Coalesce(pos.Latitude, drop.Latitude),
		Longitude: keel.Float64Of(job.Col(finalDropLongitude)).
			Coalesce(pos.Longitude, drop.Longitude),
		FreeAt: keel.Int64Of(job.Col(finalDropEndsAt)).Coalesce(now),
	}
}

func finalDrops() *keel.CTETable {
	a := shipmentdb.Assignments.As("fd")
	m := shipmentdb.ShipmentMoves.As("fd_sm")
	st := shipmentdb.Stops.As("fd_st")
	loc := locationdb.Locations.As("fd_loc")
	endsAt := st.ScheduledWindowEnd.Coalesce(st.ScheduledWindowStart)

	return keel.CTE(
		"final_drop",
		[]string{finalDropWorker, finalDropLatitude, finalDropLongitude, finalDropEndsAt},
		a.
			Join(m, m.ID.EqCol(a.ShipmentMoveID)).
			Join(st,
				st.ShipmentMoveID.EqCol(m.ID),
				st.Status.Ne(shipmentvocab.StopStatusCanceled),
			).
			Join(loc, loc.ID.EqCol(st.LocationID)).
			DistinctOn(a.PrimaryWorkerID).
			Pick(a.PrimaryWorkerID, loc.Latitude, loc.Longitude, endsAt).
			Where(
				a.PrimaryWorkerID.IsNotNull(),
				a.ArchivedAt.IsNull(),
				m.Status.In(OpenMoveStatuses...),
			).
			Order(
				a.PrimaryWorkerID.Asc(),
				keel.Desc(endsAt).NullsLast(),
				st.Sequence.Desc(),
			),
	)
}
