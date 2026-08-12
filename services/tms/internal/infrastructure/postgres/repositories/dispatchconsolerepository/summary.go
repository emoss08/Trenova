package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const atRiskLeadSeconds = int64(4 * 3600)

const positionFreshnessSeconds = int64(4 * 3600)

const metersPerMile = 1609.344

var vehiclePositionJoin = "LEFT JOIN " + buncolgen.VehiclePositionTable.As(
	buncolgen.VehiclePositionTable.Alias) +
	" ON " + buncolgen.VehiclePositionColumns.TractorID.EqColumn(
	buncolgen.AssignmentColumns.TractorID) +
	" AND " + buncolgen.VehiclePositionColumns.OrganizationID.EqColumn(
	buncolgen.AssignmentColumns.OrganizationID) +
	" AND " + buncolgen.VehiclePositionColumns.BusinessUnitID.EqColumn(
	buncolgen.AssignmentColumns.BusinessUnitID) +
	" AND " + buncolgen.VehiclePositionColumns.RecordedAt.Gte()

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
	asnCols := buncolgen.AssignmentColumns

	dayStart := filter.DayStartUnix
	if dayStart <= 0 {
		var err error
		dayStart, err = timeutils.DayStartUnix(now, "")
		if err != nil {
			return fmt.Errorf("resolve dispatch board day start: %w", err)
		}
	}

	err := r.db.DB().NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(buncolgen.CountFilter("uncovered_moves", asnCols.ID.IsNull())).
		ColumnExpr(buncolgen.CountFilter("covered_moves", asnCols.ID.IsNotNull())).
		ColumnExpr(buncolgen.CountFilter("late_moves",
			asnCols.ID.IsNull(),
			"COALESCE(orig.window_start, 0) > 0",
			"orig.window_start < ?",
		), now).
		ColumnExpr(buncolgen.CountFilter("at_risk_moves",
			asnCols.ID.IsNull(),
			"COALESCE(orig.window_start, 0) > 0",
			"orig.window_start >= ?",
			"orig.window_start <= ?",
		), now, now+atRiskLeadSeconds).
		ColumnExpr(buncolgen.CountFilter("assigned_today",
			asnCols.ID.IsNotNull(),
			asnCols.CreatedAt.Gte(),
		), dayStart).
		Apply(averageDeadheadColumn()).
		Join(shipmentJoin).
		Join(customerJoin).
		Join(assignmentJoin).
		Join(vehiclePositionJoin, now-positionFreshnessSeconds).
		Apply(joinMoveStopEdges).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ShipmentMoveScopeTenant(sq, filter.TenantInfo)
			return applyMoveFilters(sq, filter)
		}).
		Scan(ctx, summary)
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

	counts := new(struct {
		AvailableDrivers int `bun:"available_drivers"`
		UnseatedDrivers  int `bun:"unseated_drivers"`
	})

	err := r.db.DB().NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr("COUNT(*)::int AS available_drivers").
		ColumnExpr(buncolgen.CountFilter("unseated_drivers",
			"COALESCE(oa.open_assignments, 0) = 0")).
		Join(openAssignmentLateral, bun.List(openMoveStatuses)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerScopeTenant(sq, filter.TenantInfo)
			sq = applyDriverFilters(sq, filter)
			return sq.Where(cols.AvailableForDispatch.IsTrue())
		}).
		Scan(ctx, counts)
	if err != nil {
		return fmt.Errorf("scan dispatch board driver summary: %w", err)
	}

	summary.AvailableDrivers = counts.AvailableDrivers
	summary.UnseatedDrivers = counts.UnseatedDrivers

	return nil
}

func averageDeadheadColumn() func(*bun.SelectQuery) *bun.SelectQuery {
	posCols := buncolgen.VehiclePositionColumns
	asnCols := buncolgen.AssignmentColumns

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		if !dbdialect.FromBun(q.DB()).Supports(dbdialect.CapPostGIS) {
			return q.ColumnExpr("NULL AS average_deadhead")
		}

		return q.ColumnExpr(buncolgen.Expr(
			"COALESCE(AVG(ST_DistanceSphere(ST_MakePoint({0}, {1}), "+
				"ST_MakePoint(orig.longitude, orig.latitude)) / ?) "+
				"FILTER (WHERE {2} IS NOT NULL AND {3} IS NOT NULL "+
				"AND orig.latitude IS NOT NULL AND orig.longitude IS NOT NULL), 0)::float8 "+
				"AS average_deadhead",
			posCols.Longitude, posCols.Latitude, asnCols.ID, posCols.TractorID,
		), metersPerMile)
	}
}
