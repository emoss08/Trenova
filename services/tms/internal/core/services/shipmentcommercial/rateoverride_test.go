package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Re-rating fires from stop edits, assignments and the fuel price job. None of
// those may quietly replace a rate somebody negotiated by hand.
func TestRecalculate_ManualOverrideWinsOverTheContract(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.RateOverrideAmount = decimal.NewNullDecimal(decimal.NewFromInt(1400))
	entity.RateOverrideReason = "Spot rate agreed with the customer"

	engine := StubRateEngine(t, 1000)

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	assert.True(t, decimal.NewFromInt(1400).Equal(entity.FreightChargeAmount.Decimal))
	require.NotNil(t, entity.RatingDetail)
	assert.Equal(t, string(ratequote.OutcomeManualOverride), entity.RatingDetail.Source)
	assert.Contains(t, entity.RatingDetail.Explanation, "Spot rate agreed with the customer")
}

// The difference between what was charged and what the contract said is the
// rate leakage report, and it only stays true if it is stored when it happens.
func TestRecalculate_OverrideRecordsTheForegoneContractAmount(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.RateOverrideAmount = decimal.NewNullDecimal(decimal.NewFromInt(900))

	contractAmount := decimal.NewFromInt(1000)
	quote := &ratequote.RateQuote{
		Outcome:        ratequote.OutcomeRated,
		LinehaulAmount: contractAmount,
		TotalAmount:    contractAmount,
		RatedAt:        ratedAt,
	}

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:   contractAmount,
			Currency: "USD",
			Outcome:  ratequote.OutcomeRated,
			Quote:    quote,
		}, nil).
		Once()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	assert.True(t, decimal.NewFromInt(900).Equal(entity.FreightChargeAmount.Decimal))
	assert.Equal(t, ratequote.OutcomeManualOverride, quote.Outcome)
	require.True(t, quote.ForegoneAmount.Valid)
	assert.True(t, decimal.NewFromInt(100).Equal(quote.ForegoneAmount.Decimal),
		"the contract wanted 1000 and we billed 900, so 100 was given away")
}

// An invoiced shipment has numbers the customer has already seen. Nothing about
// it is recomputed, and the rate engine is not consulted at all.
func TestRecalculate_LockedShipmentKeepsItsCharges(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.RateLocked = true
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1234))

	engine := mocks.NewMockRateEngine(t)

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	assert.True(t, decimal.NewFromInt(1234).Equal(entity.FreightChargeAmount.Decimal))
}
