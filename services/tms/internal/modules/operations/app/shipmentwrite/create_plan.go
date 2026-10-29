package shipmentwrite

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
)

type createPreparation struct {
	rating     *shipmentcommercial.ContractRating
	advisories []*errortypes.AdvisoryError
}

func ratingOutcome(
	rating *shipmentcommercial.ContractRating,
) *shipmentserviceports.ShipmentRatingOutcome {
	if rating == nil || rating.Quote == nil || !rating.Quote.Outcome.Priced() {
		return nil
	}

	outcome := &shipmentserviceports.ShipmentRatingOutcome{
		Adopted:     rating.Adopted,
		Amount:      rating.ContractAmount,
		Currency:    quoteCurrency(rating.Quote),
		Explanation: rating.Quote.Explanation(),
	}
	if rating.Quote.Trace != nil {
		if winner := rating.Quote.Trace.Winner(); winner != nil {
			outcome.AgreementName = winner.AgreementName
		}
	}

	return outcome
}

func quoteCurrency(quote *ratequote.RateQuote) string {
	return stringutils.FirstNonEmpty(
		quote.BillingCurrency,
		quote.Currency,
		money.DefaultCurrencyCode,
	)
}
