package costingrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActualsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type actualsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewActuals(p ActualsParams) repositories.CostingActualsRepository {
	return &actualsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.costing-actuals-repository"),
	}
}

func (r *actualsRepository) SumExpenseByAccounts(
	ctx context.Context,
	req *repositories.SumExpenseByAccountsRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	if len(req.GLAccountIDs) == 0 {
		return map[pulid.ID]decimal.Decimal{}, nil
	}

	log := r.l.With(
		zap.String("operation", "SumExpenseByAccounts"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	var rows []struct {
		GLAccountID pulid.ID        `bun:"gl_account_id"`
		NetExpense  decimal.Decimal `bun:"net_expense"`
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("gl_account_balances_by_period AS gb").
		ColumnExpr("gb.gl_account_id").
		ColumnExpr(
			"(COALESCE(SUM(gb.period_debit_minor - gb.period_credit_minor), 0) / 100.0)::numeric AS net_expense",
		).
		Join("JOIN fiscal_periods AS fp ON fp.id = gb.fiscal_period_id AND fp.organization_id = gb.organization_id AND fp.business_unit_id = gb.business_unit_id").
		Where("gb.organization_id = ?", req.TenantInfo.OrgID).
		Where("gb.business_unit_id = ?", req.TenantInfo.BuID).
		Where("gb.gl_account_id IN (?)", bun.List(req.GLAccountIDs)).
		Where("fp.start_date >= ?", req.FromDate).
		Where("fp.end_date <= ?", req.ToDate).
		GroupExpr("gb.gl_account_id").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to sum expenses by GL account", zap.Error(err))
		return nil, err
	}

	sums := make(map[pulid.ID]decimal.Decimal, len(rows))
	for _, row := range rows {
		sums[row.GLAccountID] = row.NetExpense
	}

	return sums, nil
}

func (r *actualsRepository) ShipmentMilesByIDs(
	ctx context.Context,
	req *repositories.ShipmentMilesByIDsRequest,
) ([]*repositories.ShipmentMilesRow, error) {
	if len(req.ShipmentIDs) == 0 {
		return []*repositories.ShipmentMilesRow{}, nil
	}

	log := r.l.With(
		zap.String("operation", "ShipmentMilesByIDs"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	rows := make([]*repositories.ShipmentMilesRow, 0, len(req.ShipmentIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("sp.id AS shipment_id").
		ColumnExpr("sp.total_charge_amount AS revenue").
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE sm.loaded AND sm.distance IS NOT NULL), 0)::float8 AS loaded_miles",
		).
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE NOT sm.loaded AND sm.distance IS NOT NULL), 0)::float8 AS deadhead_miles",
		).
		ColumnExpr(
			"COUNT(*) FILTER (WHERE sm.id IS NOT NULL AND sm.distance IS NULL) > 0 AS missing_distance",
		).
		Join("LEFT JOIN shipment_moves AS sm ON sm.shipment_id = sp.id AND sm.organization_id = sp.organization_id AND sm.business_unit_id = sp.business_unit_id AND sm.status != ?", shipment.MoveStatusCanceled).
		Where("sp.organization_id = ?", req.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sp.id IN (?)", bun.List(req.ShipmentIDs)).
		GroupExpr("sp.id, sp.total_charge_amount").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to load shipment miles", zap.Error(err))
		return nil, err
	}

	return rows, nil
}

func (r *actualsRepository) FleetMiles(
	ctx context.Context,
	req *repositories.FleetMilesRequest,
) (*repositories.FleetMilesResult, error) {
	log := r.l.With(
		zap.String("operation", "FleetMiles"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	result := new(repositories.FleetMilesResult)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("shipment_moves AS sm").
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE sm.loaded), 0)::float8 AS loaded_miles",
		).
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE NOT sm.loaded), 0)::float8 AS deadhead_miles",
		).
		ColumnExpr("COALESCE(SUM(sm.distance), 0)::float8 AS total_miles").
		Join("JOIN shipments AS sp ON sp.id = sm.shipment_id AND sp.organization_id = sm.organization_id AND sp.business_unit_id = sm.business_unit_id").
		Where("sm.organization_id = ?", req.TenantInfo.OrgID).
		Where("sm.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sm.distance IS NOT NULL").
		Where("sm.distance > 0").
		Where("sm.status != ?", shipment.MoveStatusCanceled).
		Where("sp.status != ?", shipment.StatusCanceled).
		Where("sm.created_at >= ?", req.FromDate).
		Where("sm.created_at <= ?", req.ToDate).
		Scan(ctx, result)
	if err != nil {
		log.Error("failed to aggregate fleet miles", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *actualsRepository) FleetCostAggregates(
	ctx context.Context,
	req *repositories.FleetCostAggregatesRequest,
) (*repositories.FleetCostAggregatesResult, error) {
	log := r.l.With(
		zap.String("operation", "FleetCostAggregates"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	costMilesExpr := "s.total_miles"
	if !req.IncludeDeadheadMiles {
		costMilesExpr = "s.loaded_miles"
	}

	shipmentAggregates := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("sp.id").
		ColumnExpr("sp.total_charge_amount AS revenue").
		ColumnExpr("COALESCE(SUM(sm.distance), 0)::float8 AS total_miles").
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE sm.loaded), 0)::float8 AS loaded_miles",
		).
		ColumnExpr(
			"COALESCE(SUM(sm.distance) FILTER (WHERE NOT sm.loaded), 0)::float8 AS deadhead_miles",
		).
		Join("LEFT JOIN shipment_moves AS sm ON sm.shipment_id = sp.id AND sm.organization_id = sp.organization_id AND sm.business_unit_id = sp.business_unit_id AND sm.distance IS NOT NULL AND sm.distance > 0 AND sm.status != ?", shipment.MoveStatusCanceled).
		Where("sp.organization_id = ?", req.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sp.status != ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", req.FromDate).
		Where("sp.created_at <= ?", req.ToDate).
		GroupExpr("sp.id, sp.total_charge_amount")

	result := new(repositories.FleetCostAggregatesResult)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("(?) AS s", shipmentAggregates).
		ColumnExpr("COUNT(*) FILTER (WHERE s.total_miles > 0)::int AS shipment_count").
		ColumnExpr(
			"COUNT(*) FILTER (WHERE s.total_miles > 0 AND s.revenue < ? * "+costMilesExpr+")::int AS unprofitable_count",
			req.CostPerMile,
		).
		ColumnExpr(
			"COALESCE(SUM(s.revenue) FILTER (WHERE s.total_miles > 0), 0)::numeric AS total_revenue",
		).
		ColumnExpr("COALESCE(SUM(s.total_miles), 0)::float8 AS total_miles").
		ColumnExpr("COALESCE(SUM(s.loaded_miles), 0)::float8 AS loaded_miles").
		ColumnExpr("COALESCE(SUM(s.deadhead_miles), 0)::float8 AS deadhead_miles").
		Scan(ctx, result)
	if err != nil {
		log.Error("failed to aggregate fleet cost metrics", zap.Error(err))
		return nil, err
	}

	return result, nil
}
