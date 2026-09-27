package accountingsyncservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type documentRate struct {
	label          string
	currency       string
	stamp          decimal.NullDecimal
	stampDate      *int64
	documentDate   int64
	accountingDate int64
	persist        func(ctx context.Context, rate decimal.Decimal, date int64) error
}

func (s *Service) exchangeRate(
	ctx context.Context,
	sess *pushSession,
	spec *documentRate,
) (decimal.Decimal, error) {
	home := strings.ToUpper(strings.TrimSpace(sess.conn.ExternalHomeCurrency))
	code := strings.ToUpper(strings.TrimSpace(spec.currency))
	if home == "" || code == "" || code == home {
		return decimal.Zero, nil
	}
	functional := ""
	if sess.control != nil {
		functional = strings.ToUpper(strings.TrimSpace(sess.control.FunctionalCurrencyCode))
	}
	stampFits := functional == home
	if stampFits && spec.stamp.Valid && spec.stamp.Decimal.IsPositive() {
		return spec.stamp.Decimal, nil
	}

	date := exchangeratestamp.QuoteDate(
		s.policy,
		sess.control,
		spec.documentDate,
		spec.accountingDate,
	)
	if stampFits && spec.stampDate != nil {
		date = timeutils.DayStartUTC(*spec.stampDate)
	}
	day := timeutils.FormatCalendarDate(date, time.UTC)
	missing := func() error {
		return blocked(
			accountingsync.SyncErrorCurrency,
			spec.label+" is in "+code+" but "+sess.providerName+" keeps its books in "+home+
				", and Trenova has no "+code+" to "+home+" exchange rate for "+day,
			"Set up OANDA exchange rates in Trenova and retry this record, or skip it",
		)
	}
	if s.rates == nil {
		return decimal.Zero, missing()
	}
	result, err := s.rates.Convert(
		ctx,
		sess.tenant,
		code,
		home,
		decimal.NewFromInt(1),
		time.Unix(date, 0).UTC(),
	)
	if err != nil {
		s.l.Warn("exchange rate lookup failed for accounting sync", zap.Error(err))
		return decimal.Zero, missing()
	}
	if result == nil || !result.Rate.IsPositive() {
		return decimal.Zero, missing()
	}
	if stampFits && spec.persist != nil {
		if err = spec.persist(ctx, result.Rate, date); err != nil {
			return decimal.Zero, err
		}
	}
	return result.Rate, nil
}

func (s *Service) stampInvoice(
	sess *pushSession,
	id pulid.ID,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return s.invoices.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
			TenantInfo: sess.tenant,
			ID:         id,
			Rate:       rate,
			Date:       date,
		})
	}
}

func (s *Service) stampPayment(
	sess *pushSession,
	id pulid.ID,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return s.payments.StampExchangeRate(ctx, &repositories.StampExchangeRateRequest{
			TenantInfo: sess.tenant,
			ID:         id,
			Rate:       rate,
			Date:       date,
		})
	}
}

func (s *Service) stampPayable(
	sess *pushSession,
	settlement *repositories.PayableSettlement,
	paid bool,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return s.payables.StampExchangeRate(ctx, &repositories.StampPayableExchangeRateRequest{
			TenantInfo: sess.tenant,
			Kind:       settlement.Kind,
			ID:         settlement.ID,
			Paid:       paid,
			Rate:       rate,
			Date:       date,
		})
	}
}
