package rateagreement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func nullDec(value string) decimal.NullDecimal {
	if value == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(dec(value))
}

func weightBreak(from, to, rate string) *rateagreement.RateAgreementRuleBreak {
	return &rateagreement.RateAgreementRuleBreak{
		ID:         pulid.MustNew("ragb_"),
		FromWeight: dec(from),
		ToWeight:   nullDec(to),
		Rate:       dec(rate),
	}
}

// A class tariff gets cheaper per hundredweight as freight gets heavier, so a
// shipment near the top of a band can cost more than the same freight declared
// at the bottom of the next one. Carriers let the shipper pay the lower of the
// two.
func TestDeficitRatingBumpsToTheCheaperHeavierBreak(t *testing.T) {
	t.Parallel()

	breaks := []*rateagreement.RateAgreementRuleBreak{
		weightBreak("0", "1000", "30"),
		weightBreak("1000", "5000", "20"),
		weightBreak("5000", "", "10"),
	}

	// 4,900 lb at $20/cwt is $980. Declaring 5,000 lb at $10/cwt is $500.
	result, err := rateagreement.ApplyDeficitRating(breaks, dec("4900"), true)

	require.NoError(t, err)
	assert.True(t, result.Bumped)
	assert.True(t, result.Charge.Equal(dec("500")), "charge was %s", result.Charge)
	assert.True(t, result.RatedWeight.Equal(dec("5000")))
	assert.True(t, result.ActualWeight.Equal(dec("4900")))
	assert.Equal(t, breaks[2].ID, result.UsedBreak.ID)
	assert.Equal(t, breaks[1].ID, result.ActualBreak.ID)
}

func TestDeficitRatingLeavesTheShipmentAloneWhenItIsAlreadyCheapest(t *testing.T) {
	t.Parallel()

	breaks := []*rateagreement.RateAgreementRuleBreak{
		weightBreak("0", "1000", "30"),
		weightBreak("1000", "5000", "20"),
		weightBreak("5000", "", "10"),
	}

	// 1,100 lb at $20/cwt is $220; bumping to 5,000 lb at $10/cwt would be $500.
	result, err := rateagreement.ApplyDeficitRating(breaks, dec("1100"), true)

	require.NoError(t, err)
	assert.False(t, result.Bumped)
	assert.True(t, result.Charge.Equal(dec("220")), "charge was %s", result.Charge)
	assert.Equal(t, breaks[1].ID, result.UsedBreak.ID)
}

// With three or more bands the cheapest is not always the adjacent one, so
// every heavier band has to be considered rather than just the next.
func TestDeficitRatingSkipsPastAnAdjacentBreakToACheaperOne(t *testing.T) {
	t.Parallel()

	breaks := []*rateagreement.RateAgreementRuleBreak{
		weightBreak("0", "1000", "40"),
		weightBreak("1000", "2000", "39"),
		weightBreak("2000", "20000", "2"),
	}

	// 900 lb at $40/cwt is $360. The next band up at 1,000 lb and $39/cwt is
	// $390 — worse. Two bands up, 2,000 lb at $2/cwt is $40.
	result, err := rateagreement.ApplyDeficitRating(breaks, dec("900"), true)

	require.NoError(t, err)
	assert.True(t, result.Bumped)
	assert.True(t, result.Charge.Equal(dec("40")), "charge was %s", result.Charge)
	assert.Equal(t, breaks[2].ID, result.UsedBreak.ID)
}

func TestDeficitRatingCanBeTurnedOff(t *testing.T) {
	t.Parallel()

	breaks := []*rateagreement.RateAgreementRuleBreak{
		weightBreak("0", "1000", "30"),
		weightBreak("1000", "5000", "20"),
		weightBreak("5000", "", "10"),
	}

	result, err := rateagreement.ApplyDeficitRating(breaks, dec("4900"), false)

	require.NoError(t, err)
	assert.False(t, result.Bumped)
	assert.True(t, result.Charge.Equal(dec("980")), "charge was %s", result.Charge)
}

func TestBreakMinimumChargeFloorsTheBand(t *testing.T) {
	t.Parallel()

	light := weightBreak("0", "1000", "30")
	light.MinCharge = nullDec("150")

	result, err := rateagreement.ApplyDeficitRating(
		[]*rateagreement.RateAgreementRuleBreak{light},
		dec("100"),
		true,
	)

	require.NoError(t, err)
	// 100 lb at $30/cwt is $30, floored to the band's $150 minimum.
	assert.True(t, result.Charge.Equal(dec("150")), "charge was %s", result.Charge)
}

func TestDeficitRatingReportsWhenNoBreakCoversTheWeight(t *testing.T) {
	t.Parallel()

	_, err := rateagreement.ApplyDeficitRating(
		[]*rateagreement.RateAgreementRuleBreak{weightBreak("1000", "5000", "20")},
		dec("100"),
		true,
	)

	require.ErrorIs(t, err, rateagreement.ErrNoBreakForWeight)
}

func TestDensityIsWeightOverVolume(t *testing.T) {
	t.Parallel()

	density, ok := rateagreement.Density(dec("820"), dec("100"))

	require.True(t, ok)
	assert.True(t, density.Equal(dec("8.2")), "density was %s", density)
}

// Dimensions go missing on entry all the time. A rule that wanted density has
// to fall back to the commodity's class rather than refuse the shipment.
func TestDensityIsUnavailableWithoutVolume(t *testing.T) {
	t.Parallel()

	_, ok := rateagreement.Density(dec("820"), decimal.Zero)
	assert.False(t, ok)

	_, ok = rateagreement.Density(decimal.Zero, dec("100"))
	assert.False(t, ok)
}
