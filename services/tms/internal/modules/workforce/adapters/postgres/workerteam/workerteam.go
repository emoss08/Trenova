package workerteam

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/fleetcodecol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type Scope struct {
	WorkerID       buncolgen.Column
	OrganizationID buncolgen.Column
	BusinessUnitID buncolgen.Column
	ManagerIDs     []pulid.ID
}

func Where(db bun.IDB, scope *Scope) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		if len(scope.ManagerIDs) == 0 {
			return sq
		}
		return sq.Where(scope.WorkerID.Qualified()+" IN (?)", Members(db, scope))
	}
}

func Members(db bun.IDB, scope *Scope) *bun.SelectQuery {
	wrk := workercol.WorkerColumns
	fleet := fleetcodecol.FleetCodeColumns

	return db.NewSelect().
		Model((*worker.Worker)(nil)).
		Column(wrk.ID.Bare()).
		Where(wrk.OrganizationID.EqColumn(scope.OrganizationID)).
		Where(wrk.BusinessUnitID.EqColumn(scope.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(wrk.ManagerID.In(), bun.List(scope.ManagerIDs)).
				WhereOr(
					wrk.FleetCodeID.Qualified()+" IN (?)",
					db.NewSelect().
						Model((*fleetcode.FleetCode)(nil)).
						Column(fleet.ID.Bare()).
						Where(fleet.OrganizationID.EqColumn(wrk.OrganizationID)).
						Where(fleet.BusinessUnitID.EqColumn(wrk.BusinessUnitID)).
						Where(fleet.ManagerID.In(), bun.List(scope.ManagerIDs)),
				)
		})
}
