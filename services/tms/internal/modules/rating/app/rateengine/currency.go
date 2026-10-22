package rateengine

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

func (s *Service) exchangeFor(
	ctx context.Context,
	rateCtx *RateContext,
	currency string,
	trace *ratetypes.Trace,
) (*rules.TariffExchange, error) {
	if currency == "" || currency == rateCtx.BillingCurrency {
		return nil, nil
	}

	result, err := s.exchange.Convert(
		ctx,
		rateCtx.TenantInfo,
		currency,
		rateCtx.BillingCurrency,
		decimal.NewFromInt(1),
		time.Unix(rateCtx.AsOf, 0).UTC(),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"No exchange rate from {0} to {1} was available on the rating date, so the shipment cannot be rated",
			currency,
			rateCtx.BillingCurrency,
		).WithInternal(err)
	}

	trace.FX = &ratetypes.FXConversion{
		FromCurrency: result.FromCurrency,
		ToCurrency:   result.ToCurrency,
		Rate:         result.Rate,
		RateDate:     result.Date,
	}

	return &rules.TariffExchange{Rate: result.Rate, Currency: rateCtx.BillingCurrency}, nil
}
