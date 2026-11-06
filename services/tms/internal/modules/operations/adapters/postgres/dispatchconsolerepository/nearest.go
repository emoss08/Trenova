package dispatchconsolerepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/geoutils"
)

func nearestFirst(
	q keel.Query[*worker.Worker],
	w *workerdb.WorkersTable,
	wt *tractordb.TractorsTable,
	near geoutils.Point,
	now int64,
) (keel.Query[*worker.Worker], keel.Expr) {
	job := finalDrops()
	pos := telematicsdb.VehiclePositions.As("npos")
	ec := equipmentcontinuitydb.EquipmentContinuities.As("ndrop")
	drop := locationdb.Locations.As("ndrop_loc")

	q = q.
		With(job).
		LeftJoin(job, job.Col(finalDropWorker).EqCol(w.ID)).
		LeftJoin(pos,
			pos.TractorID.EqCol(wt.ID),
			pos.RecordedAt.Gte(now-positionFreshnessSeconds),
		).
		LeftJoin(ec,
			ec.EquipmentID.EqCol(wt.ID),
			ec.EquipmentType.Eq(equipmentcontinuity.EquipmentTypeTractor),
			ec.IsCurrent.IsTrue(),
		).
		LeftJoin(drop, drop.ID.EqCol(ec.CurrentLocationID))

	return q, postgis.DistanceSphere(
		postgis.Pt(
			keel.Float64Of(job.Col(finalDropLongitude)).Coalesce(pos.Longitude, drop.Longitude),
			keel.Float64Of(job.Col(finalDropLatitude)).Coalesce(pos.Latitude, drop.Latitude),
		),
		postgis.Pt(near.Longitude, near.Latitude),
	)
}

const (
	finalDropWorker    = "worker_id"
	finalDropLatitude  = "latitude"
	finalDropLongitude = "longitude"
)

func finalDrops() *keel.CTETable {
	a := shipmentdb.Assignments.As("fd")
	m := shipmentdb.ShipmentMoves.As("fd_sm")
	st := shipmentdb.Stops.As("fd_st")
	loc := locationdb.Locations.As("fd_loc")

	return keel.CTE(
		"final_drop",
		[]string{finalDropWorker, finalDropLatitude, finalDropLongitude},
		a.
			Join(m, m.ID.EqCol(a.ShipmentMoveID)).
			Join(st,
				st.ShipmentMoveID.EqCol(m.ID),
				st.Status.Ne(shipmentvocab.StopStatusCanceled),
			).
			Join(loc, loc.ID.EqCol(st.LocationID)).
			DistinctOn(a.PrimaryWorkerID).
			Pick(a.PrimaryWorkerID, loc.Latitude, loc.Longitude).
			Where(
				a.PrimaryWorkerID.IsNotNull(),
				a.ArchivedAt.IsNull(),
				m.Status.In(openMoveStatuses...),
			).
			Order(
				a.PrimaryWorkerID.Asc(),
				keel.Desc(st.ScheduledWindowEnd.Coalesce(st.ScheduledWindowStart)).NullsLast(),
				st.Sequence.Desc(),
			),
	)
}
