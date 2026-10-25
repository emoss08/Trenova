package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const defaultTeamPageSize = 500

// Headcount is grouped SQL over the roster rather than a walk of it. Counting
// a few thousand drivers in Go to answer "how many in each terminal" would be
// the wrong shape at any size.

type headcountGroup struct {
	key   keel.Expr
	label keel.Expr
	code  keel.Expr
	color keel.Expr
	group []keel.Expr
}

// rosterCount is the shared shape of the three headcount groupings: active and
// terminated counted together so a terminal that has emptied out still shows,
// and the driving half broken out because that is the number a safety director
// asks for.
func (r *repository) rosterCount(
	ctx context.Context,
	operation string,
	g *headcountGroup,
) ([]workerports.HeadcountRow, error) {
	w := workerdb.Workers
	jp := w.Position()
	active := w.Status.Eq(domaintypes.StatusActive)
	workers := keel.Count().Filter(active)

	rows, err := keel.Into[workerports.HeadcountRow](ctx, r.db.Keel(ctx), w.
		Group(g.group...).
		Pick(
			g.key,
			g.label,
			g.code,
			g.color,
			workers,
			keel.Count().Filter(keel.And(
				active,
				keel.DynOf(jp.IsDrivingPosition).Coalesce(true),
			)),
			keel.Count().Filter(w.Status.Ne(domaintypes.StatusActive)),
			keel.DynLit(0),
		).
		Order(workers.Desc(), keel.Asc(g.label)))
	if err != nil {
		r.l.Error("failed to count headcount by "+operation, zap.Error(err))
		return nil, fmt.Errorf("count headcount by %s: %w", operation, err)
	}

	return rows, nil
}

func (r *repository) HeadcountByFleet(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]workerports.HeadcountRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.HeadcountRow, error) {
		w := workerdb.Workers
		fc := w.FleetCode()
		return r.rosterCount(ctx, "fleet", &headcountGroup{
			key:   orEmpty(w.FleetCodeID),
			label: keel.DynOf(fc.Description).Coalesce(fc.Code, "No terminal"),
			code:  orEmpty(fc.Code),
			color: orEmpty(fc.Color),
			group: []keel.Expr{w.FleetCodeID, fc.Code, fc.Description, fc.Color},
		})
	})
}

func (r *repository) HeadcountByPosition(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]workerports.HeadcountRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.HeadcountRow, error) {
		w := workerdb.Workers
		jp := w.Position()
		return r.rosterCount(ctx, "position", &headcountGroup{
			key:   orEmpty(w.PositionID),
			label: keel.DynOf(jp.Title).Coalesce("No position"),
			code:  orEmpty(jp.Code),
			color: keel.DynLit(""),
			group: []keel.Expr{w.PositionID, jp.Title, jp.Code},
		})
	})
}

func (r *repository) HeadcountByDepartment(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]workerports.HeadcountRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.HeadcountRow, error) {
		jp := workerdb.Workers.Position()
		return r.rosterCount(ctx, "department", &headcountGroup{
			key:   asText(jp.Department).Coalesce(""),
			label: asText(jp.Department).Coalesce("Unassigned"),
			code:  keel.DynLit(""),
			color: keel.DynLit(""),
			group: []keel.Expr{jp.Department},
		})
	})
}

// teamScope is the predicate behind "my team": workers who name one of these
// users as their manager, or who sit in a terminal one of them runs. Both
// paths are needed — a carrier that never fills in a worker's manager still
// has terminal managers, and one that does still wants the terminal covered.
func teamScope(managerIDs []pulid.ID) keel.Pred {
	w := workerdb.Workers
	return keel.Or(w.ManagerID.In(managerIDs...), w.FleetCode().ManagerID.In(managerIDs...))
}

func (r *repository) TeamMembers(
	ctx context.Context,
	req *workerports.TeamScopeRequest,
) ([]workerports.TeamMemberRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.TeamMemberRow, error) {
		if len(req.ManagerIDs) == 0 {
			return []workerports.TeamMemberRow{}, nil
		}

		w := workerdb.Workers
		fc := w.FleetCode()
		jp := w.Position()
		p := w.Profile()
		direct := keel.DynOf(w.ManagerID.In(req.ManagerIDs...))

		rows, err := keel.Into[workerports.TeamMemberRow](ctx, r.db.Keel(ctx), w.
			Where(
				p.WorkerID.IsNotNull(),
				teamScope(req.ManagerIDs),
				keel.When(!req.IncludeInactive, w.Status.Eq(domaintypes.StatusActive)),
			).
			Pick(
				w.ID,
				w.FirstName,
				w.LastName,
				asText(w.Status),
				orEmpty(w.FleetCodeID),
				orEmpty(fc.Code),
				orEmpty(fc.Color),
				orEmpty(w.PositionID),
				orEmpty(jp.Title),
				orEmpty(w.ManagerID),
				direct,
				asText(p.ComplianceStatus),
				asText(p.TrainingHealth),
				asText(p.SafetyRating),
				p.HireDate,
				p.TerminationDate,
			).
			Order(keel.Desc(direct), w.LastName.Asc(), w.FirstName.Asc()).
			Limit(limitOr(req.Limit, defaultTeamPageSize)))
		if err != nil {
			r.l.Error("failed to read team members", zap.Error(err))
			return nil, fmt.Errorf("read team members: %w", err)
		}

		return rows, nil
	})
}

// ManagesWorker answers the authorization question directly. A manager of four
// hundred drivers must not pull four hundred rows to approve one day of leave.
func (r *repository) ManagesWorker(
	ctx context.Context,
	_ pagination.TenantInfo,
	managerIDs []pulid.ID,
	workerID pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		if len(managerIDs) == 0 || workerID.IsNil() {
			return false, nil
		}

		w := workerdb.Workers
		exists, err := w.Where(w.ID.Eq(workerID), teamScope(managerIDs)).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check manager scope", zap.Error(err))
			return false, fmt.Errorf("check manager scope: %w", err)
		}

		return exists, nil
	})
}
