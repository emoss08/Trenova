package exchangeraterepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.ExchangeRateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.exchange-rate-repository"),
	}
}

func (r *repository) GetRate(
	ctx context.Context,
	req *repositories.GetExchangeRateRequest,
) (*exchangerate.ExchangeRate, error) {
	entity := new(exchangerate.ExchangeRate)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("er.organization_id = ?", req.TenantInfo.OrgID).
		Where("er.business_unit_id = ?", req.TenantInfo.BuID).
		Where("er.from_currency = ?", req.FromCurrency).
		Where("er.to_currency = ?", req.ToCurrency).
		Where("er.date = ?", req.Date.Format("2006-01-02")).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpsertRates(
	ctx context.Context,
	req *repositories.UpsertExchangeRatesRequest,
) error {
	return r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		for idx := range req.Rates {
			rate := req.Rates[idx]
			_, err := tx.NewInsert().
				Model(rate).
				On("CONFLICT (organization_id, business_unit_id, from_currency, to_currency, date) DO UPDATE").
				Set("rate = EXCLUDED.rate").
				Set("fetched_at = EXCLUDED.fetched_at").
				Exec(txCtx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) GetLatestDate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Time, error) {
	var latestDate time.Time
	err := r.db.DB().
		NewSelect().
		Model((*exchangerate.ExchangeRate)(nil)).
		ColumnExpr("er.date").
		Where("er.organization_id = ?", tenantInfo.OrgID).
		Where("er.business_unit_id = ?", tenantInfo.BuID).
		Order("er.date DESC").
		Limit(1).
		Scan(ctx, &latestDate)
	if err != nil {
		return nil, err
	}
	return &latestDate, nil
}
