package costingrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

func NewActuals(p ActualsParams) costingcontrolports.CostingActualsRepository {
	return &actualsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.costing-actuals-repository"),
	}
}

type expenseRow struct {
	GLAccountID pulid.ID
	NetExpense  decimal.Decimal
}

func floatSum(e keel.Expr) keel.Expr {
	return keel.Raw("COALESCE(?, 0)::float8", e)
}

func (r *actualsRepository) SumExpenseByAccounts(
	ctx context.Context,
	req *costingcontrolports.SumExpenseByAccountsRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]decimal.Decimal, error) {
		if len(req.GLAccountIDs) == 0 {
			return map[pulid.ID]decimal.Decimal{}, nil
		}

		gb := glbalancedb.GLAccountPeriodBalances
		fp := fiscalperioddb.FiscalPeriods
		rows, err := keel.Into[expenseRow](ctx, r.db.Keel(ctx), gb.
			Join(fp, fp.ID.EqCol(gb.FiscalPeriodID)).
			Pick(
				gb.GLAccountID,
				keel.Raw(
					"(COALESCE(SUM(? - ?), 0) / 100.0)::numeric",
					gb.PeriodDebitMinor,
					gb.PeriodCreditMinor,
				),
			).
			Where(
				gb.GLAccountID.In(req.GLAccountIDs...),
				fp.StartDate.Gte(req.FromDate),
				fp.EndDate.Lte(req.ToDate),
			).
			Group(gb.GLAccountID))
		if err != nil {
			r.l.Error(
				"failed to sum expenses by GL account",
				zap.String("orgID", req.TenantInfo.OrgID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		sums := make(map[pulid.ID]decimal.Decimal, len(rows))
		for _, row := range rows {
			sums[row.GLAccountID] = row.NetExpense
		}

		return sums, nil
	})
}

func (r *actualsRepository) ShipmentMilesByIDs(
	ctx context.Context,
	req *costingcontrolports.ShipmentMilesByIDsRequest,
) ([]*costingcontrolports.ShipmentMilesRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*costingcontrolports.ShipmentMilesRow, error) {
			if len(req.ShipmentIDs) == 0 {
				return []*costingcontrolports.ShipmentMilesRow{}, nil
			}

			sp := shipmentdb.Shipments
			sm := shipmentdb.ShipmentMoves.As("sm")
			rows, err := keel.Into[costingcontrolports.ShipmentMilesRow](ctx, r.db.Keel(ctx), sp.
				LeftJoin(
					sm,
					sm.ShipmentID.EqCol(sp.ID),
					sm.Status.Ne(shipment.MoveStatusCanceled),
				).
				Pick(
					sp.ID,
					floatSum(sm.Distance.Sum().Filter(keel.And(sm.Loaded.IsTrue(), sm.Distance.IsNotNull()))),
					floatSum(sm.Distance.Sum().Filter(keel.And(sm.Loaded.IsFalse(), sm.Distance.IsNotNull()))),
					keel.Raw("COALESCE(?, 0)", sp.TotalChargeAmount),
					keel.Raw("? > 0", keel.Count().Filter(keel.And(sm.ID.IsNotNull(), sm.Distance.IsNull()))),
				).
				Where(sp.ID.In(req.ShipmentIDs...)).
				Group(sp.ID, sp.TotalChargeAmount))
			if err != nil {
				r.l.Error(
					"failed to load shipment miles",
					zap.String("orgID", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			out := make([]*costingcontrolports.ShipmentMilesRow, 0, len(rows))
			for i := range rows {
				out = append(out, &rows[i])
			}
			return out, nil
		},
	)
}

func (r *actualsRepository) FleetMiles(
	ctx context.Context,
	req *costingcontrolports.FleetMilesRequest,
) (*costingcontrolports.FleetMilesResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*costingcontrolports.FleetMilesResult, error) {
			sm := shipmentdb.ShipmentMoves
			sp := shipmentdb.Shipments.As("sp")
			result, err := keel.IntoOne[costingcontrolports.FleetMilesResult](ctx, r.db.Keel(ctx), sm.
				Join(sp, sp.ID.EqCol(sm.ShipmentID)).
				Pick(
					floatSum(sm.Distance.Sum().Filter(sm.Loaded.IsTrue())),
					floatSum(sm.Distance.Sum().Filter(sm.Loaded.IsFalse())),
					floatSum(sm.Distance.Sum()),
				).
				Where(
					sm.Distance.IsNotNull(),
					sm.Distance.Gt(0),
					sm.Status.Ne(shipment.MoveStatusCanceled),
					sp.Status.Ne(shipment.StatusCanceled),
					sm.CreatedAt.Gte(req.FromDate),
					sm.CreatedAt.Lte(req.ToDate),
				))
			if err != nil {
				r.l.Error(
					"failed to aggregate fleet miles",
					zap.String("orgID", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return &result, nil
		},
	)
}

func (r *actualsRepository) FleetCostAggregates(
	ctx context.Context,
	req *costingcontrolports.FleetCostAggregatesRequest,
) (*costingcontrolports.FleetCostAggregatesResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*costingcontrolports.FleetCostAggregatesResult, error) {
			sp := shipmentdb.Shipments
			sm := shipmentdb.ShipmentMoves.As("sm")
			totals := keel.CTE(
				"shipment_totals",
				[]string{"id", "revenue", "total_miles", "loaded_miles", "deadhead_miles"},
				sp.
					LeftJoin(
						sm,
						sm.ShipmentID.EqCol(sp.ID),
						sm.Distance.IsNotNull(),
						sm.Distance.Gt(0),
						sm.Status.Ne(shipment.MoveStatusCanceled),
					).
					Pick(
						sp.ID,
						sp.TotalChargeAmount,
						floatSum(sm.Distance.Sum()),
						floatSum(sm.Distance.Sum().Filter(sm.Loaded.IsTrue())),
						floatSum(sm.Distance.Sum().Filter(sm.Loaded.IsFalse())),
					).
					Where(
						sp.Status.Ne(shipment.StatusCanceled),
						sp.CreatedAt.Gte(req.FromDate),
						sp.CreatedAt.Lte(req.ToDate),
					).
					Group(sp.ID, sp.TotalChargeAmount),
			)

			costMiles := totals.Col("total_miles")
			if !req.IncludeDeadheadMiles {
				costMiles = totals.Col("loaded_miles")
			}
			moving := totals.Col("total_miles").Gt(0)

			result, err := keel.IntoOne[costingcontrolports.FleetCostAggregatesResult](
				ctx,
				r.db.Keel(ctx),
				totals.Query().With(totals).Pick(
					keel.Raw("(?)::int", keel.Count().Filter(moving)),
					keel.Raw("(?)::int", keel.Count().Filter(keel.And(
						moving,
						keel.Raw("? < ?::numeric * ?", totals.Col("revenue"), req.CostPerMile, costMiles),
					))),
					keel.Raw("COALESCE(?, 0)::numeric", keel.Sum(totals.Col("revenue")).Filter(moving)),
					floatSum(keel.Sum(totals.Col("total_miles"))),
					floatSum(keel.Sum(totals.Col("loaded_miles"))),
					floatSum(keel.Sum(totals.Col("deadhead_miles"))),
				),
			)
			if err != nil {
				r.l.Error(
					"failed to aggregate fleet cost metrics",
					zap.String("orgID", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return &result, nil
		},
	)
}
