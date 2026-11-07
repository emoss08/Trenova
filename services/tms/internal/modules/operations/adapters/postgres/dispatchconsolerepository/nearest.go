package dispatchconsolerepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/driverstart"
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
	q, start := driverstart.Join(q, w, wt, now)

	return q, postgis.DistanceSphere(
		postgis.Pt(start.Longitude, start.Latitude),
		postgis.Pt(near.Longitude, near.Latitude),
	)
}
