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
		m := dispatchmarketdb.MarketValues
		rows, err := m.Where(m.Cell.In(req.Cells...)).
			Limit(len(req.Cells)*int(dispatchmarket.SlotsPerWeek)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list market values by cell: %w", err)
		}

		return rows, nil
	})
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
