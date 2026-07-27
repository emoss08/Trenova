package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// atRiskLeadSeconds is how close a pickup appointment has to be, with the move still
// uncovered, before the console calls it at risk.
const atRiskLeadSeconds = int64(4 * 3600)

// GetBoardSummary produces the metrics strip. Move-side and driver-side counts are two
// aggregate queries rather than a scan of the rows the board already returned, so the
// numbers stay honest even when the board itself is truncated by its limit.
func (r *repository) GetBoardSummary(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
) (*repositories.BoardSummary, error) {
	summary := new(repositories.BoardSummary)
	now := timeutils.NowUnix()

	if err := r.scanMoveSummary(ctx, filter, now, summary); err != nil {
		return nil, err
	}
	if err := r.scanDriverSummary(ctx, filter, summary); err != nil {
		return nil, err
	}

	if summary.AvailableDrivers > 0 {
		seated := summary.AvailableDrivers - summary.UnseatedDrivers
		summary.UtilizationPct = float64(seated) / float64(summary.AvailableDrivers) * 100
	}

	return summary, nil
}

func (r *repository) scanMoveSummary(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
	now int64,
	summary *repositories.BoardSummary,
) error {
	moveCols := buncolgen.ShipmentMoveColumns
	asnCols := buncolgen.AssignmentColumns

	const windowJoin = `LEFT JOIN LATERAL (
		SELECT MIN(stp.scheduled_window_start) AS window_start
		FROM stops AS stp
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> 'Canceled'
	) AS win ON TRUE`

	err := r.db.DB().NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr("COUNT(*) FILTER (WHERE a.id IS NULL)::int AS uncovered_moves").
		ColumnExpr("COUNT(*) FILTER (WHERE a.id IS NOT NULL)::int AS covered_moves").
		ColumnExpr("COUNT(*) FILTER (WHERE a.id IS NULL " +
			"AND COALESCE(win.window_start, 0) > 0 " +
			"AND win.window_start < ?)::int AS late_moves").
		ColumnExpr("COUNT(*) FILTER (WHERE a.id IS NULL " +
			"AND COALESCE(win.window_start, 0) > 0 " +
			"AND win.window_start >= ? AND win.window_start <= ?)::int AS at_risk_moves").
		ColumnExpr("COUNT(*) FILTER (WHERE a.id IS NOT NULL AND a.created_at >= ?)::int " +
			"AS assigned_today").
		Join("LEFT JOIN assignments AS a ON " + asnCols.ShipmentMoveID.EqColumn(moveCols.ID) +
			" AND " + asnCols.ArchivedAt.IsNull()).
		Join(windowJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ShipmentMoveScopeTenant(sq, filter.TenantInfo).
				Where(moveCols.Status.In(), bun.List([]shipment.MoveStatus{
					shipment.MoveStatusNew,
					shipment.MoveStatusAssigned,
					shipment.MoveStatusInTransit,
				}))
			if filter.WindowEnd > 0 {
				sq = sq.Where("COALESCE(win.window_start, 0) <= ?", filter.WindowEnd)
			}
			return sq
		}).
		Scan(ctx, summary, now, now, now+atRiskLeadSeconds, startOfDay(now))
	if err != nil {
		return fmt.Errorf("scan dispatch board move summary: %w", err)
	}

	return nil
}

func (r *repository) scanDriverSummary(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
	summary *repositories.BoardSummary,
) error {
	cols := buncolgen.WorkerColumns

	const openAssignmentJoin = `LEFT JOIN LATERAL (
		SELECT COUNT(*)::int AS open_assignments
		FROM assignments AS a
		JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id
		WHERE a.primary_worker_id = wrk.id
			AND a.organization_id = wrk.organization_id
			AND a.business_unit_id = wrk.business_unit_id
			AND a.archived_at IS NULL
			AND sm.status IN ('Assigned', 'InTransit')
	) AS oa ON TRUE`

	counts := new(struct {
		AvailableDrivers int `bun:"available_drivers"`
		UnseatedDrivers  int `bun:"unseated_drivers"`
	})

	err := r.db.DB().NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr("COUNT(*)::int AS available_drivers").
		ColumnExpr("COUNT(*) FILTER (WHERE COALESCE(oa.open_assignments, 0) = 0)::int " +
			"AS unseated_drivers").
		Join(openAssignmentJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerScopeTenant(sq, filter.TenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive).
				Where(cols.CanBeAssigned.IsTrue()).
				Where(cols.AvailableForDispatch.IsTrue())
			if len(filter.FleetCodeIDs) > 0 {
				sq = sq.Where(cols.FleetCodeID.In(), bun.List(filter.FleetCodeIDs))
			}
			return sq
		}).
		Scan(ctx, counts)
	if err != nil {
		return fmt.Errorf("scan dispatch board driver summary: %w", err)
	}

	summary.AvailableDrivers = counts.AvailableDrivers
	summary.UnseatedDrivers = counts.UnseatedDrivers

	return nil
}

func startOfDay(ts int64) int64 {
	return ts - (ts % 86400)
}
