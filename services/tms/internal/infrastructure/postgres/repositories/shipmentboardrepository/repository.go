package shipmentboardrepository

import (
	"context"
	"errors"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrScopeMissing = errors.New("shipment board scope is required")

const (
	countAlias         = "count"
	stageRankAlias     = "stage_rank"
	stageRevenueAlias  = "revenue"
	quickCountAliasPfx = "qf_"
)

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

func NewBoardRepository(r *Repository) repositories.ShipmentBoardRepository { return r }

func NewWatchlistRepository(r *Repository) repositories.ShipmentWatchlistRepository { return r }

func (r *Repository) scoped(
	dba bun.IDB,
	scope *repositories.ShipmentBoardScope,
) (*bun.SelectQuery, error) {
	if scope == nil {
		return nil, ErrScopeMissing
	}

	return shipmentrepository.ApplyAggregateScope(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		scope.Filter,
		&scope.Options,
	)
}

func (r *Repository) StageSummary(
	ctx context.Context,
	scope *repositories.ShipmentBoardScope,
) ([]*repositories.ShipmentStageSummaryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.ShipmentStageSummaryRow, error) {
			sp := buncolgen.ShipmentColumns

			q, err := r.scoped(r.db.DBForContext(ctx), scope)
			if err != nil {
				return nil, err
			}

			rows := make([]*repositories.ShipmentStageSummaryRow, 0, len(shipment.Stages()))
			if err = q.
				ColumnExpr(sp.StageRank.As(stageRankAlias)).
				ColumnExpr(buncolgen.Count(countAlias)).
				ColumnExpr(sp.TotalChargeAmount.Expr("COALESCE(SUM({}), 0) AS "+stageRevenueAlias)).
				GroupExpr(sp.StageRank.Qualified()).
				OrderExpr(sp.StageRank.OrderAsc()).
				Scan(ctx, &rows); err != nil {
				r.l.Error("failed to summarize shipment stages", zap.Error(err))
				return nil, err
			}

			return rows, nil
		},
	)
}

func (r *Repository) QuickFilterTotals(
	ctx context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) ([]repositories.ShipmentQuickFilterTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]repositories.ShipmentQuickFilterTotal, error) {
			totals := make([]repositories.ShipmentQuickFilterTotal, len(req.Filters))
			if len(req.Filters) == 0 {
				return totals, nil
			}
			if req.Scope == nil {
				return nil, ErrScopeMissing
			}

			dba := r.db.DBForContext(ctx)
			q, err := r.scoped(dba, req.Scope)
			if err != nil {
				return nil, err
			}

			revenue := buncolgen.ShipmentColumns.TotalChargeAmount.Qualified()
			dest := make([]any, 0, len(req.Filters)*2)
			for i, spec := range req.Filters {
				cond, cErr := shipmentrepository.QuickFilterCondition(
					dba,
					req.Scope.Options.QuickFilterBasis,
					spec,
				)
				if cErr != nil {
					return nil, cErr
				}
				alias := quickCountAliasPfx + strconv.Itoa(i)
				q = q.
					ColumnExpr("COUNT(*) FILTER (WHERE ?) AS ?", cond, bun.Ident(alias+"_count")).
					ColumnExpr(
						"COALESCE(SUM("+revenue+") FILTER (WHERE ?), 0) AS ?",
						cond,
						bun.Ident(alias+"_revenue"),
					)
				dest = append(dest, &totals[i].Count, &totals[i].Revenue)
			}

			if err = q.Scan(ctx, dest...); err != nil {
				r.l.Error("failed to total shipment quick filters", zap.Error(err))
				return nil, err
			}

			return totals, nil
		},
	)
}
