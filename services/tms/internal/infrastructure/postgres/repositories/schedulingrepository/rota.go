package schedulingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultRotaPageSize = 300
	secondsPerDay       = 86400
)

// The rota is four grouped queries over a week rather than a walk of the
// roster: a fifty-driver week is four round trips, not two hundred.
//
// Time off and leave come back as the ranges they are stored as and are
// expanded into days by the composer. Expanding them in SQL would need
// generate_series, which is Postgres-only, and a week is seven days — the
// expansion is not the expensive part.

// dayStartExpr truncates an epoch column to its UTC midnight. The column is
// already an integer number of seconds, so this is integer arithmetic rather
// than a date function, and it means the same thing in both dialects.
func dayStartExpr(column string) string {
	return fmt.Sprintf("((%s / %d) * %d)", column, secondsPerDay, secondsPerDay)
}

func (r *repository) rosterScope(
	ctx context.Context,
	req *repositories.RotaQuery,
) *bun.SelectQuery {
	cols := buncolgen.WorkerColumns
	q := buncolgen.WorkerScopeTenant(
		r.db.DBForContext(ctx).NewSelect().Model((*worker.Worker)(nil)),
		req.TenantInfo,
	).Where(cols.Status.Eq(), domaintypes.StatusActive)
	if !req.FleetCodeID.IsNil() {
		q = q.Where(cols.FleetCodeID.Eq(), req.FleetCodeID)
	}
	if len(req.ManagerIDs) > 0 {
		fleet := buncolgen.FleetCodeColumns
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.ManagerID.In(), bun.List(req.ManagerIDs)).
				WhereOr(
					cols.FleetCodeID.Qualified()+" IN (?)",
					r.db.DBForContext(ctx).
						NewSelect().
						Model((*fleetcode.FleetCode)(nil)).
						Column(fleet.ID.Bare()).
						Where(fleet.OrganizationID.EqColumn(cols.OrganizationID)).
						Where(fleet.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
						Where(fleet.ManagerID.In(), bun.List(req.ManagerIDs)),
				)
		})
	}
	return q
}

// RotaWorkers is the roster line: who is on the board, and which pattern they
// are on for the week. The assignment in force is joined rather than listed
// separately so the week comes back in one row per worker.
func (r *repository) RotaWorkers(
	ctx context.Context,
	req *repositories.RotaQuery,
) ([]repositories.RotaWorkerRow, error) {
	rows := make([]repositories.RotaWorkerRow, 0, 32)

	if err := r.rosterScope(ctx, req).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		Join("LEFT JOIN worker_shift_assignments AS wsa").
		JoinOn("wsa.worker_id = wrk.id").
		JoinOn("wsa.organization_id = wrk.organization_id").
		JoinOn("wsa.business_unit_id = wrk.business_unit_id").
		JoinOn("wsa.effective_from <= ?", req.WeekEnd).
		JoinOn("(wsa.effective_to IS NULL OR wsa.effective_to >= ?)", req.WeekStart).
		ColumnExpr("wrk.id AS worker_id").
		ColumnExpr("wrk.first_name AS first_name").
		ColumnExpr("wrk.last_name AS last_name").
		ColumnExpr("COALESCE(fc.code, '') AS fleet_code").
		ColumnExpr("COALESCE(fc.color, '') AS fleet_color").
		ColumnExpr("COALESCE(wsa.shift_template_id, '') AS shift_template_id").
		ColumnExpr("COALESCE(wsa.cycle_offset_weeks, 0) AS cycle_offset_weeks").
		OrderExpr("wrk.last_name, wrk.first_name").
		Limit(limitOr(req.Limit, defaultRotaPageSize)).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read rota workers", zap.Error(err))
		return nil, fmt.Errorf("read rota workers: %w", err)
	}

	return rows, nil
}

// RotaTimeOffRanges is the approved time off overlapping the week. Ranges
// rather than days: the composer expands them, and seven days of expansion is
// not worth a Postgres-only query.
func (r *repository) RotaTimeOffRanges(
	ctx context.Context,
	req *repositories.RotaQuery,
) ([]repositories.RotaRangeRow, error) {
	rows := make([]repositories.RotaRangeRow, 0, 16)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTO)(nil)).
		With("roster", r.rosterScope(ctx, req).ColumnExpr("wrk.id AS worker_id")).
		Join("JOIN roster").
		JoinOn("roster.worker_id = wpto.worker_id").
		Where("wpto.organization_id = ?", req.TenantInfo.OrgID).
		Where("wpto.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wpto.status = ?", "Approved").
		Where("wpto.start_date < ?", req.WeekEnd).
		Where("wpto.end_date >= ?", req.WeekStart).
		ColumnExpr("wpto.worker_id AS worker_id").
		ColumnExpr("wpto.start_date AS starts_at").
		ColumnExpr("wpto.end_date AS ends_at").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read rota time off", zap.Error(err))
		return nil, fmt.Errorf("read rota time off: %w", err)
	}

	return rows, nil
}

// RotaLeaveRanges is the same over open leave cases. A case with no end date
// runs on, so it comes back with a zero end and the composer treats it as
// covering the rest of the week.
func (r *repository) RotaLeaveRanges(
	ctx context.Context,
	req *repositories.RotaQuery,
) ([]repositories.RotaRangeRow, error) {
	rows := make([]repositories.RotaRangeRow, 0, 16)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerLeaveCase)(nil)).
		With("roster", r.rosterScope(ctx, req).ColumnExpr("wrk.id AS worker_id")).
		Join("JOIN roster").
		JoinOn("roster.worker_id = wlc.worker_id").
		Where("wlc.organization_id = ?", req.TenantInfo.OrgID).
		Where("wlc.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wlc.status = ?", "Approved").
		Where("wlc.starts_at < ?", req.WeekEnd).
		Where("(wlc.ends_at IS NULL OR wlc.ends_at >= ?)", req.WeekStart).
		ColumnExpr("wlc.worker_id AS worker_id").
		ColumnExpr("wlc.starts_at AS starts_at").
		ColumnExpr("COALESCE(wlc.ends_at, 0) AS ends_at").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read rota leave", zap.Error(err))
		return nil, fmt.Errorf("read rota leave: %w", err)
	}

	return rows, nil
}

// RotaAssignedDays counts the work dispatch has already put on each day. The
// day is truncated with integer arithmetic on the epoch column, which means
// the same thing in both dialects.
func (r *repository) RotaAssignedDays(
	ctx context.Context,
	req *repositories.RotaQuery,
) ([]repositories.RotaDayRow, error) {
	rows := make([]repositories.RotaDayRow, 0, 32)
	dayStart := dayStartExpr("asn.created_at")

	if err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("assignments AS asn").
		With("roster", r.rosterScope(ctx, req).ColumnExpr("wrk.id AS worker_id")).
		Join("JOIN roster").
		JoinOn("roster.worker_id = asn.primary_worker_id").
		Where("asn.organization_id = ?", req.TenantInfo.OrgID).
		Where("asn.business_unit_id = ?", req.TenantInfo.BuID).
		Where("asn.archived_at IS NULL").
		Where(dayStart+" >= ?", req.WeekStart).
		Where(dayStart+" < ?", req.WeekEnd).
		ColumnExpr("asn.primary_worker_id AS worker_id").
		ColumnExpr(dayStart+" AS day_start").
		ColumnExpr("COUNT(*) AS count").
		GroupExpr("asn.primary_worker_id, "+dayStart).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read rota assignments", zap.Error(err))
		return nil, fmt.Errorf("read rota assignments: %w", err)
	}

	return rows, nil
}
