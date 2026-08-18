package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chargeIDPtr() *pulid.ID {
	id := pulid.MustNew("acc_")

	return &id
}

func money(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func ratedBuySide(amount string) *services.RatedShipment {
	quoteID := pulid.MustNew("rqt_")
	agreementID := pulid.MustNew("rag_")

	return &services.RatedShipment{
		Amount:      money(amount),
		Currency:    "USD",
		Outcome:     ratequote.OutcomeRated,
		AgreementID: &agreementID,
		Quote: &ratequote.RateQuote{
			ID:      quoteID,
			Outcome: ratequote.OutcomeRated,
		},
	}
}

// A contract that priced the load produces one number, and the quote explains
// how it got there. Splitting it back into a rate and a distance would let the
// assignment recompute a total that disagrees with its own quote, which is the
// exact failure contract pricing exists to remove.
func TestBuySideRating_PricesTheAssignmentAsAFlatAmount(t *testing.T) {
	t.Parallel()

	rating, err := buySideRating(ratedBuySide("1450.75"), nil, decimal.NullDecimal{})

	require.NoError(t, err)
	assert.Equal(t, shipment.CarrierRateMethodFlat, rating.RateMethod)
	assert.True(t, rating.BaseRate.Equal(money("1450.75")), "base rate was %s", rating.BaseRate)
}

func TestBuySideRating_CarriesTheQuoteSoThePayIsAnswerable(t *testing.T) {
	t.Parallel()

	rated := ratedBuySide("1450")

	rating, err := buySideRating(rated, nil, decimal.NullDecimal{})

	require.NoError(t, err)
	require.NotNil(t, rating.RateQuoteID)
	assert.Equal(t, rated.Quote.ID, *rating.RateQuoteID)
}

// A lane no carrier contract covers is not something to guess at. Writing a
// zero would make the load look free and settle for nothing.
func TestBuySideRating_RefusesWhenNoContractCoversTheLane(t *testing.T) {
	t.Parallel()

	rated := ratedBuySide("0")
	rated.Outcome = ratequote.OutcomeNoRateFound

	_, err := buySideRating(rated, nil, decimal.NullDecimal{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "carrier")
}

// The contract's accessorial schedule is the same list the sell side reads, so
// the rate confirmation and the carrier settlement cannot disagree about what
// this carrier is owed.
func TestBuySideRating_TakesAccessorialsFromTheCarriersContract(t *testing.T) {
	t.Parallel()

	chargeID := pulid.MustNew("acc_")
	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{
			{
				AccessorialChargeID: chargeID,
				AutoApply:           true,
				Method:              accessorialcharge.MethodFlat,
				Amount:              money("125"),
				AccessorialCharge:   &accessorialcharge.AccessorialCharge{ID: chargeID, Code: "LUMP"},
			},
		},
	}

	rating, err := buySideRating(ratedBuySide("1450"), agreement, decimal.NullDecimal{})

	require.NoError(t, err)
	require.Len(t, rating.Accessorials, 1)
	require.NotNil(t, rating.Accessorials[0].AccessorialChargeID)
	assert.Equal(t, chargeID, *rating.Accessorials[0].AccessorialChargeID)
	assert.True(t, rating.Accessorials[0].Amount.Equal(money("125")))
}

// A row waiting for somebody to add it by hand is a price, not a charge.
func TestBuySideRating_LeavesManualAccessorialsAlone(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{
			{AccessorialChargeID: pulid.MustNew("acc_"), AutoApply: false, Amount: money("125")},
		},
	}

	rating, err := buySideRating(ratedBuySide("1450"), agreement, decimal.NullDecimal{})

	require.NoError(t, err)
	assert.Empty(t, rating.Accessorials)
}

// An accessorial whose application depends on a formula cannot be answered here
// — this path has no engine — and applying it anyway would pay a carrier for a
// condition nobody checked.
func TestBuySideRating_SkipsAccessorialsWithAnUncheckedCondition(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{
			{
				AccessorialChargeID: pulid.MustNew("acc_"),
				AutoApply:           true,
				ApplyCondition:      "totalStops > 2",
				Amount:              money("125"),
			},
		},
	}

	rating, err := buySideRating(ratedBuySide("1450"), agreement, decimal.NullDecimal{})

	require.NoError(t, err)
	assert.Empty(t, rating.Accessorials)
}

// Margin is measured against what the load sold for, which is the number a
// broker is actually managing.
func TestBuySideRating_MeasuresMarginAgainstTheSellPrice(t *testing.T) {
	t.Parallel()

	rating, err := buySideRating(
		ratedBuySide("1400"),
		nil,
		decimal.NewNullDecimal(money("2000")),
	)

	require.NoError(t, err)
	assert.True(t, rating.Margin.Amount.Equal(money("600")),
		"margin was %s", rating.Margin.Amount)
	assert.True(t, rating.Margin.Percent.Equal(money("30")),
		"percent was %s", rating.Margin.Percent)
}

func TestBuySideRating_FlagsAMarginBelowTheContractsFloor(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		MarginFloorPercent: decimal.NewNullDecimal(money("15")),
	}

	rating, err := buySideRating(
		ratedBuySide("1900"),
		agreement,
		decimal.NewNullDecimal(money("2000")),
	)

	require.NoError(t, err)
	assert.True(t, rating.Margin.BelowFloor)
}

// A shipment nobody has priced for the customer yet is not a margin breach.
// A broker quoting cost first is ordinary.
func TestBuySideRating_UnpricedSellSideIsNotAMarginBreach(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		MarginFloorPercent: decimal.NewNullDecimal(money("15")),
	}

	rating, err := buySideRating(ratedBuySide("1900"), agreement, decimal.NullDecimal{})

	require.NoError(t, err)
	assert.False(t, rating.Margin.Breached())
}

func appliedRequest() *repositories.AssignMoveToCarrierRequest {
	return &repositories.AssignMoveToCarrierRequest{
		CarrierID: pulid.MustNew("car_"),
		AutoRate:  true,
	}
}

// Auto-rating fills the request the same way a person typing the numbers would,
// so everything downstream — validation, totals, the rate confirmation — runs
// on one path rather than two.
func TestApplyBuySideRating_FillsTheRequestTheWayAPersonWould(t *testing.T) {
	t.Parallel()

	req := appliedRequest()
	quoteID := pulid.MustNew("rqt_")

	applyBuySideRating(req, &buySideResult{
		RateMethod:    shipment.CarrierRateMethodFlat,
		BaseRate:      money("1450"),
		FuelSurcharge: money("210"),
		RateQuoteID:   &quoteID,
		Accessorials: []repositories.CarrierAccessorialInput{
			{AccessorialChargeID: chargeIDPtr(), Amount: money("125")},
		},
	})

	assert.Equal(t, shipment.CarrierRateMethodFlat, req.RateMethod)
	assert.True(t, req.BaseRate.Equal(money("1450")))
	assert.True(t, req.FuelSurcharge.Equal(money("210")))
	assert.Len(t, req.Accessorials, 1)
}

// Somebody who typed a rate meant it. Auto-rating over the top of it would be
// the system overruling a negotiated number, which is what makes people switch
// auto-rating off.
func TestApplyBuySideRating_IsNotAskedForWhenARateWasTyped(t *testing.T) {
	t.Parallel()

	req := &repositories.AssignMoveToCarrierRequest{
		CarrierID: pulid.MustNew("car_"),
		AutoRate:  true,
		BaseRate:  money("1600"),
	}

	assert.False(t, shouldAutoRate(req), "a typed rate is a decision, not a gap")
}

func TestShouldAutoRate_IsSkippedWhenNotAskedFor(t *testing.T) {
	t.Parallel()

	assert.False(t, shouldAutoRate(&repositories.AssignMoveToCarrierRequest{}))
}

func TestShouldAutoRate_AppliesWhenAskedForAndNoRateWasTyped(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldAutoRate(appliedRequest()))
}

// A guardrail the organization turned on has to actually stop the assignment,
// or it is decoration.
func TestEnforceMarginFloor_BlocksABreachWhenEnforcementIsOn(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{BelowFloor: true, Explanation: "margin of 5% is below the 15% this contract requires"},
		true,
		false,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "15%")
}

// With enforcement off the breach is still worth knowing about, but it is not
// the system's decision to make.
func TestEnforceMarginFloor_AllowsABreachWhenEnforcementIsOff(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{BelowFloor: true, Explanation: "below floor"},
		false,
		false,
	)

	assert.NoError(t, err)
}

// The override is how somebody says "I know, do it anyway", mirroring the
// insurance warning this service already has.
func TestEnforceMarginFloor_AnExplicitOverrideGoesThrough(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{BelowFloor: true, Explanation: "below floor"},
		true,
		true,
	)

	assert.NoError(t, err)
}

func TestEnforceMarginFloor_NothingBreachedNeedsNoOverride(t *testing.T) {
	t.Parallel()

	assert.NoError(t, enforceMarginFloor(ratetypes.MarginVerdict{}, true, false))
}
