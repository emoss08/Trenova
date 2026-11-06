package dispatchmarketrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket/dispatchmarketdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dispatchmarketports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-market-repository"),
	}
}

func (r *repository) ListByCells(
	ctx context.Context,
	req *dispatchmarketports.ListByCellsRequest,
) ([]*dispatchmarket.MarketValue, error) {
	if len(req.Cells) == 0 {
		return []*dispatchmarket.MarketValue{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*dispatchmarket.MarketValue, error) {
		return listed(cellsQuery(req.Cells).List(ctx, r.db.Keel(ctx)))
	})
}

func (r *repository) LockByCells(
	ctx context.Context,
	req *dispatchmarketports.ListByCellsRequest,
) ([]*dispatchmarket.MarketValue, error) {
	if len(req.Cells) == 0 {
		return []*dispatchmarket.MarketValue{}, nil
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*dispatchmarket.MarketValue, error) {
		m := dispatchmarketdb.MarketValues
		return listed(cellsQuery(req.Cells).
			Order(m.Cell.Asc(), m.Slot.Asc()).
			Lock(keel.ForUpdate).
			List(ctx, r.db.Keel(ctx)))
	})
}

func cellsQuery(cells []int64) keel.Query[*dispatchmarket.MarketValue] {
	m := dispatchmarketdb.MarketValues

	return m.Where(m.Cell.In(cells...)).Limit(len(cells) * int(dispatchmarket.SlotsPerWeek))
}

func listed(rows []*dispatchmarket.MarketValue, err error) ([]*dispatchmarket.MarketValue, error) {
	if err != nil {
		return nil, fmt.Errorf("list market values by cell: %w", err)
	}

	return rows, nil
}

func (r *repository) Upsert(ctx context.Context, values []*dispatchmarket.MarketValue) error {
	if len(values) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		m := dispatchmarketdb.MarketValues
		if _, err := m.Upsert(
			ctx,
			r.db.Keel(ctx),
			values,
			keel.OnConflict(m.OrganizationID, m.BusinessUnitID, m.Cell, m.Slot).Update(
				m.ValueCents,
				m.Weeks,
				m.WeekIndex,
				m.WeekSumCents,
				m.WeekSamples,
				m.Observations,
				m.ObservedAt,
			),
		); err != nil {
			return fmt.Errorf("upsert market values: %w", err)
		}

		return nil
	})
}
