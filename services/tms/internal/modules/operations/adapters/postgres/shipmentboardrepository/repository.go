package shipmentboardrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrScopeMissing = errors.New("shipment board scope is required")

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type Repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) *Repository {
	return &Repository{db: p.DB, l: p.Logger.Named("postgres.shipment-board-repository")}
}

func NewBoardRepository(r *Repository) shipmentports.ShipmentBoardRepository { return r }

func NewWatchlistRepository(r *Repository) shipmentports.ShipmentWatchlistRepository { return r }

func scoped(scope *shipmentports.ShipmentBoardScope) (*keel.Query[shipment.Shipment], error) {
	if scope == nil {
		return nil, ErrScopeMissing
	}

	return shipmentrepository.AggregateScope(scope.Filter, &scope.Options)
}

func revenueSum() keel.Expr {
	return keel.DynOf(shipmentdb.Shipments.TotalChargeAmount).Sum().Coalesce(0)
}

func (r *Repository) StageSummary(
	ctx context.Context,
	scope *shipmentports.ShipmentBoardScope,
) ([]*shipmentports.ShipmentStageSummaryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentStageSummaryRow, error) {
			q, err := scoped(scope)
			if err != nil {
				return nil, err
			}

			sp := shipmentdb.Shipments
			rows, err := keel.Into[shipmentports.ShipmentStageSummaryRow](
				ctx,
				r.db.Keel(ctx),
				q.Group(sp.StageRank).
					Pick(sp.StageRank, keel.Count(), revenueSum()).
					Order(sp.StageRank.Asc()),
			)
			if err != nil {
				r.l.Error("failed to summarize shipment stages", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

type quickFilterTotalsRow struct {
	Counts   []int64
	Revenues []string
}

func (r *Repository) QuickFilterTotals(
	ctx context.Context,
	req *shipmentports.CountShipmentQuickFiltersRequest,
) ([]shipmentports.ShipmentQuickFilterTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]shipmentports.ShipmentQuickFilterTotal, error) {
			totals := make([]shipmentports.ShipmentQuickFilterTotal, len(req.Filters))
			if len(req.Filters) == 0 {
				return totals, nil
			}
			if req.Scope == nil {
				return nil, ErrScopeMissing
			}

			q, err := scoped(req.Scope)
			if err != nil {
				return nil, err
			}

			totalsQuery, err := quickFilterTotalsQuery(
				q,
				req.Scope.Options.QuickFilterBasis,
				req.Filters,
			)
			if err != nil {
				return nil, err
			}

			row, err := keel.IntoOne[quickFilterTotalsRow](ctx, r.db.Keel(ctx), totalsQuery)
			if err != nil {
				r.l.Error("failed to total shipment quick filters", zap.Error(err))
				return nil, err
			}

			for i := range totals {
				if i < len(row.Counts) {
					totals[i].Count = int(row.Counts[i])
				}
				if i < len(row.Revenues) {
					amount, pErr := decimal.NewFromString(row.Revenues[i])
					if pErr != nil {
						return nil, fmt.Errorf("parse quick filter revenue: %w", pErr)
					}
					totals[i].Revenue = amount
				}
			}

			return totals, nil
		},
	)
}

func quickFilterTotalsQuery(
	q *keel.Query[shipment.Shipment],
	basis *shipmentports.ShipmentQuickFilterBasis,
	filters []shipment.QuickFilterSpec,
) (*keel.Query[shipment.Shipment], error) {
	counts := make([]any, 0, len(filters))
	revenues := make([]any, 0, len(filters))
	revenue := keel.DynOf(shipmentdb.Shipments.TotalChargeAmount)
	for _, spec := range filters {
		cond, err := shipmentrepository.QuickFilterCondition(basis, spec)
		if err != nil {
			return nil, err
		}
		counts = append(counts, keel.Count().Filter(cond))
		revenues = append(revenues, revenue.Sum().Filter(cond).Coalesce(0).Cast(keel.TypeString))
	}

	return q.Pick(keel.DynArray(counts...), keel.DynArray(revenues...)), nil
}

func pointers[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}
