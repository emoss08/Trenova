package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

// ratedAt is the fixed moment the stubbed quotes claim to have been produced,
// so a test asserting on a rating detail is never comparing against the clock.
const ratedAt = int64(7300)

// StubRateEngine returns a rate engine that always prices at the given amount.
//
// The calculator's job is to assemble a shipment's totals from a linehaul,
// detention, contract accessorials and fuel. Which of those produced the
// linehaul is the engine's business, so tests of the calculator hold it fixed
// and vary everything around it.
func StubRateEngine(t *testing.T, amount int64) *mocks.MockRateEngine {
	t.Helper()

	return StubRateEngineForAgreement(t, amount, nil)
}

// StubRateEngineForAgreement is StubRateEngine for the cases that care which
// contract won, because the engine — not the caller — is what stamps the
// agreement onto the shipment.
func StubRateEngineForAgreement(
	t *testing.T,
	amount int64,
	agreementID *pulid.ID,
) *mocks.MockRateEngine {
	t.Helper()

	linehaul := decimal.NewFromInt(amount)

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:      linehaul,
			Currency:    "USD",
			Outcome:     ratequote.OutcomeRated,
			AgreementID: agreementID,
			Quote: &ratequote.RateQuote{
				Outcome:         ratequote.OutcomeRated,
				RateAgreementID: agreementID,
				LinehaulAmount:  linehaul,
				TotalAmount:     linehaul,
				RatedAt:         ratedAt,
				Trace: &ratetypes.Trace{
					Totals: ratetypes.Totals{Linehaul: linehaul, Total: linehaul},
				},
			},
		}, nil).
		Maybe()

	return engine
}
