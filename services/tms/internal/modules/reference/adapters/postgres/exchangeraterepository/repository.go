package exchangeraterepository

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/exchangerate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/exchangerate/exchangeratedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	exchangeRateErrors    = dberror.Mapping{Entity: "ExchangeRate"}
	settlementQuoteErrors = dberror.Mapping{Entity: "SettlementQuote"}
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

func New(p Params) exchangerateports.ExchangeRateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.exchange-rate-repository"),
	}
}

func (r *repository) GetRate(
	ctx context.Context,
	req *exchangerateports.GetExchangeRateRequest,
) (*exchangerate.ExchangeRate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*exchangerate.ExchangeRate, error) {
		er := exchangeratedb.ExchangeRates
		entity, err := er.Where(
			er.Provider.Eq(req.Provider),
			er.FromCurrency.Eq(req.FromCurrency),
			er.ToCurrency.Eq(req.ToCurrency),
			er.RateType.Eq(req.RateType),
			er.Date.Eq(req.Date.Format(time.DateOnly)),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, exchangeRateErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) UpsertRates(
	ctx context.Context,
	req *exchangerateports.UpsertExchangeRatesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.Rates) == 0 {
			return nil
		}

		er := exchangeratedb.ExchangeRates
		_, err := er.Upsert(ctx, r.db.Keel(ctx), req.Rates,
			keel.OnConflict(
				er.OrganizationID,
				er.BusinessUnitID,
				er.Provider,
				er.FromCurrency,
				er.ToCurrency,
				er.RateType,
				er.Date,
			).Update(
				er.Bid,
				er.Ask,
				er.Mid,
				er.SelectedRate,
				er.SourceTimestamp,
				er.FetchedAt,
				er.SettlementEligible,
			))
		if err != nil {
			r.l.Error("failed to upsert exchange rates", zap.Error(err))
			return exchangeRateErrors.Map(err, "")
		}

		return nil
	})
}

func (r *repository) CreateSettlementQuote(
	ctx context.Context,
	req *exchangerateports.CreateSettlementQuoteQuery,
) (*exchangerate.SettlementQuote, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*exchangerate.SettlementQuote, error) {
		req.Quote.OrganizationID = req.TenantInfo.OrgID
		req.Quote.BusinessUnitID = req.TenantInfo.BuID

		if err := exchangeratedb.SettlementQuotes.Create(ctx, r.db.Keel(ctx), req.Quote); err != nil {
			r.l.Error("failed to create settlement quote", zap.Error(err))
			return nil, settlementQuoteErrors.Map(err, req.Quote.ID.String())
		}

		return req.Quote, nil
	})
}

func (r *repository) GetLatestDate(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*time.Time, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*time.Time, error) {
		er := exchangeratedb.ExchangeRates
		latest, err := keel.IntoOne[time.Time](ctx, r.db.Keel(ctx),
			er.Pick(er.Date).Order(er.Date.Desc()).Limit(1))
		if err != nil {
			return nil, exchangeRateErrors.Map(err, "")
		}

		return &latest, nil
	})
}
