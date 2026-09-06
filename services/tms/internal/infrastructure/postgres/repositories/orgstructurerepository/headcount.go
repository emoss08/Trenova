package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const defaultTeamPageSize = 500

// Headcount is grouped SQL over the roster rather than a walk of it. Counting
// a few thousand drivers in Go to answer "how many in each terminal" would be
// the wrong shape at any size.

// rosterCount is the shared shape of the three headcount groupings: active and
// terminated counted together so a terminal that has emptied out still shows,
// and the driving half broken out because that is the number a safety director
// asks for.
func (r *repository) rosterCount(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Join("LEFT JOIN job_positions AS jpos").
		JoinOn("jpos.id = wrk.position_id").
		JoinOn("jpos.organization_id = wrk.organization_id").
		JoinOn("jpos.business_unit_id = wrk.business_unit_id").
		Where("wrk.organization_id = ?", tenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", tenantInfo.BuID).
		ColumnExpr("COUNT(*) FILTER (WHERE wrk.status = 'Active') AS workers").
		ColumnExpr(
			"COUNT(*) FILTER (WHERE wrk.status = 'Active' AND COALESCE(jpos.is_driving_position, TRUE)) AS drivers",
		).
		ColumnExpr("COUNT(*) FILTER (WHERE wrk.status <> 'Active') AS terminated")
}

func (r *repository) HeadcountByFleet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	rows := make([]repositories.HeadcountRow, 0, 8)
	if err := r.rosterCount(ctx, tenantInfo).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		ColumnExpr("COALESCE(wrk.fleet_code_id, '') AS key").
		ColumnExpr("COALESCE(fc.description, fc.code, 'No terminal') AS label").
		ColumnExpr("COALESCE(fc.code, '') AS code").
		ColumnExpr("COALESCE(fc.color, '') AS color").
		GroupExpr("wrk.fleet_code_id, fc.code, fc.description, fc.color").
		OrderExpr("workers DESC, label").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to count headcount by fleet", zap.Error(err))
		return nil, fmt.Errorf("count headcount by fleet: %w", err)
	}
	return rows, nil
}

func (r *repository) HeadcountByPosition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	rows := make([]repositories.HeadcountRow, 0, 16)
	if err := r.rosterCount(ctx, tenantInfo).
		ColumnExpr("COALESCE(wrk.position_id, '') AS key").
		ColumnExpr("COALESCE(jpos.title, 'No position') AS label").
		ColumnExpr("COALESCE(jpos.code, '') AS code").
		ColumnExpr("'' AS color").
		GroupExpr("wrk.position_id, jpos.title, jpos.code").
		OrderExpr("workers DESC, label").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to count headcount by position", zap.Error(err))
		return nil, fmt.Errorf("count headcount by position: %w", err)
	}
	return rows, nil
}

func (r *repository) HeadcountByDepartment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.HeadcountRow, error) {
	rows := make([]repositories.HeadcountRow, 0, 9)
	if err := r.rosterCount(ctx, tenantInfo).
		ColumnExpr("COALESCE(jpos.department::text, '') AS key").
		ColumnExpr("COALESCE(jpos.department::text, 'Unassigned') AS label").
		ColumnExpr("'' AS code").
		ColumnExpr("'' AS color").
		GroupExpr("jpos.department").
		OrderExpr("workers DESC, label").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to count headcount by department", zap.Error(err))
		return nil, fmt.Errorf("count headcount by department: %w", err)
	}
	return rows, nil
}

// teamScope is the predicate behind "my team": workers who name one of these
// users as their manager, or who sit in a terminal one of them runs. Both
// paths are needed — a carrier that never fills in a worker's manager still
// has terminal managers, and one that does still wants the terminal covered.
func teamScope(managerIDs []pulid.ID) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Where("wrk.manager_id IN (?)", bun.In(managerIDs)).
			WhereOr("fc.manager_id IN (?)", bun.In(managerIDs))
	}
}

func (r *repository) TeamMembers(
	ctx context.Context,
	req *repositories.TeamScopeRequest,
) ([]repositories.TeamMemberRow, error) {
	if len(req.ManagerIDs) == 0 {
		return []repositories.TeamMemberRow{}, nil
	}

	rows := make([]repositories.TeamMemberRow, 0, 32)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Join("JOIN worker_profiles AS wrkp").
		JoinOn("wrkp.worker_id = wrk.id").
		JoinOn("wrkp.organization_id = wrk.organization_id").
		JoinOn("wrkp.business_unit_id = wrk.business_unit_id").
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		Join("LEFT JOIN job_positions AS jpos").
		JoinOn("jpos.id = wrk.position_id").
		JoinOn("jpos.organization_id = wrk.organization_id").
		JoinOn("jpos.business_unit_id = wrk.business_unit_id").
		Where("wrk.organization_id = ?", req.TenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", req.TenantInfo.BuID).
		WhereGroup(" AND ", teamScope(req.ManagerIDs)).
		ColumnExpr("wrk.id AS worker_id").
		ColumnExpr("wrk.first_name AS first_name").
		ColumnExpr("wrk.last_name AS last_name").
		ColumnExpr("wrk.status::text AS status").
		ColumnExpr("COALESCE(wrk.fleet_code_id, '') AS fleet_code_id").
		ColumnExpr("COALESCE(fc.code, '') AS fleet_code").
		ColumnExpr("COALESCE(fc.color, '') AS fleet_color").
		ColumnExpr("COALESCE(wrk.position_id, '') AS position_id").
		ColumnExpr("COALESCE(jpos.title, '') AS position_title").
		ColumnExpr("COALESCE(wrk.manager_id, '') AS manager_id").
		ColumnExpr("(wrk.manager_id IN (?)) AS direct", bun.In(req.ManagerIDs)).
		ColumnExpr("wrkp.compliance_status::text AS compliance_status").
		ColumnExpr("wrkp.training_health::text AS training_health").
		ColumnExpr("wrkp.safety_rating::text AS safety_rating").
		ColumnExpr("wrkp.hire_date AS hire_date").
		ColumnExpr("wrkp.termination_date AS termination_date").
		OrderExpr("direct DESC, wrk.last_name, wrk.first_name").
		Limit(limitOr(req.Limit, defaultTeamPageSize))

	if !req.IncludeInactive {
		q = q.Where("wrk.status = 'Active'")
	}

	if err := q.Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read team members", zap.Error(err))
		return nil, fmt.Errorf("read team members: %w", err)
	}

	return rows, nil
}

// ManagesWorker answers the authorization question directly. A manager of four
// hundred drivers must not pull four hundred rows to approve one day of leave.
func (r *repository) ManagesWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	managerIDs []pulid.ID,
	workerID pulid.ID,
) (bool, error) {
	if len(managerIDs) == 0 || workerID.IsNil() {
		return false, nil
	}

	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		Where("wrk.organization_id = ?", tenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", tenantInfo.BuID).
		Where("wrk.id = ?", workerID).
		WhereGroup(" AND ", teamScope(managerIDs)).
		Exists(ctx)
	if err != nil {
		r.l.Error("failed to check manager scope", zap.Error(err))
		return false, fmt.Errorf("check manager scope: %w", err)
	}

	return exists, nil
}
