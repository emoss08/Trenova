package accountingsyncpush

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type DocumentRate struct {
	Label          string
	Currency       string
	Stamp          decimal.NullDecimal
	StampDate      *int64
	DocumentDate   int64
	AccountingDate int64
	Persist        func(ctx context.Context, rate decimal.Decimal, date int64) error
}

func (k *Kit) ExchangeRate(
	ctx context.Context,
	sess *Session,
	spec *DocumentRate,
) (decimal.Decimal, error) {
	home := strings.ToUpper(strings.TrimSpace(sess.Conn.ExternalHomeCurrency))
	code := strings.ToUpper(strings.TrimSpace(spec.Currency))
	if home == "" || code == "" || code == home {
		return decimal.Zero, nil
	}
	functional := ""
	if sess.Control != nil {
		functional = strings.ToUpper(strings.TrimSpace(sess.Control.FunctionalCurrencyCode))
	}
	stampFits := functional == home
	if stampFits && spec.Stamp.Valid && spec.Stamp.Decimal.IsPositive() {
		return spec.Stamp.Decimal, nil
	}

	date := exchangeratestamp.QuoteDate(
		k.policy,
		sess.Control,
		spec.DocumentDate,
		spec.AccountingDate,
	)
	if stampFits && spec.StampDate != nil {
		date = timeutils.DayStartUTC(*spec.StampDate)
	}
	day := timeutils.FormatCalendarDate(date, time.UTC)
	missing := func() error {
		return Blocked(
			accountingsync.SyncErrorCurrency,
			spec.Label+" is in "+code+" but "+sess.ProviderName+" keeps its books in "+home+
				", and Trenova has no "+code+" to "+home+" exchange rate for "+day,
			"Set up OANDA exchange rates in Trenova and retry this record, or skip it",
		)
	}
	if k.rates == nil {
		return decimal.Zero, missing()
	}
	result, err := k.rates.Convert(
		ctx,
		sess.Tenant,
		code,
		home,
		decimal.NewFromInt(1),
		time.Unix(date, 0).UTC(),
	)
	if err != nil {
		k.l.Warn("exchange rate lookup failed for accounting sync", zap.Error(err))
		return decimal.Zero, missing()
	}
	if result == nil || !result.Rate.IsPositive() {
		return decimal.Zero, missing()
	}
	if stampFits && spec.Persist != nil {
		if err = spec.Persist(ctx, result.Rate, date); err != nil {
			return decimal.Zero, err
		}
	}
	return result.Rate, nil
}
