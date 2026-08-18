package rateengine

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// convert moves a priced amount from the contract's currency into the one the
// organization bills in.
//
// It happens once, at the very end, and never before the guardrails. A
// contractual minimum charge converted first would drift with the exchange
// rate, which is a breach of the contract rather than a rounding difference.
//
// The rate is looked up as of the rating date, not today, for the same reason
// the contract terms are: a shipment that moved last month is converted at last
// month's rate however long the invoice takes to produce. Both the rate and the
// date are recorded, so the quote stays reproducible after the market moves.
func (s *Service) convert(
	ctx context.Context,
	rateCtx *RateContext,
	priced priceResult,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if priced.Currency == "" || priced.Currency == rateCtx.BillingCurrency {
		return priced.Amount, nil
	}

	result, err := s.exchange.Convert(
		ctx,
		rateCtx.TenantInfo,
		priced.Currency,
		rateCtx.BillingCurrency,
		priced.Amount,
		time.Unix(rateCtx.AsOf, 0).UTC(),
	)
	if err != nil {
		// Refusing to price is worse than pricing in the contract's own
		// currency and saying so: the shipment still has to move, and an
		// invoice flagged for a missing exchange rate is a solvable problem
		// while a blocked save is not.
		s.l.Warn("failed to convert contract currency",
			zap.String("from", priced.Currency),
			zap.String("to", rateCtx.BillingCurrency),
			zap.Error(err),
		)

		trace.Warn(
			"No exchange rate was available for " + priced.Currency + " to " +
				rateCtx.BillingCurrency + " on the rating date; " +
				"the amount is stated in " + priced.Currency,
		)

		return priced.Amount, nil
	}

	trace.FX = &ratetypes.FXConversion{
		FromCurrency: result.FromCurrency,
		ToCurrency:   result.ToCurrency,
		Rate:         result.Rate,
		RateDate:     result.Date,
	}

	trace.AddComponent(ratetypes.Component{
		Kind:  ratetypes.ComponentKindCurrencyConversion,
		Label: "Currency conversion",
		Basis: priced.Amount.String() + " " + result.FromCurrency + " @ " +
			result.Rate.String(),
		Rate:   decimal.NewNullDecimal(result.Rate),
		Amount: result.Converted.Sub(priced.Amount),
		Source: ratetypes.ComponentSourceExchangeRate,
	})

	return result.Converted, nil
}
