package ratetypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func TestMargin_IsWhatIsLeftAfterTheCarrierIsPaid(t *testing.T) {
	t.Parallel()

	assert.True(t, ratetypes.Margin(dec("2000"), dec("1600")).Equal(dec("400")))
}

// A load bought for more than it sold for is a real and important case — it is
// what a margin floor exists to catch — so it comes back negative rather than
// clamped at zero.
func TestMargin_GoesNegativeWhenTheBuyExceedsTheSell(t *testing.T) {
	t.Parallel()

	assert.True(t, ratetypes.Margin(dec("1500"), dec("1600")).Equal(dec("-100")))
}

func TestMarginPercent_IsAShareOfWhatTheCustomerPays(t *testing.T) {
	t.Parallel()

	// 400 of 2000 is 20%.
	assert.True(t, ratetypes.MarginPercent(dec("2000"), dec("1600")).Equal(dec("20")))
}

func TestMarginPercent_GoesNegativeOnALoss(t *testing.T) {
	t.Parallel()

	// -100 of 1500 is -6.6667%, rounded to four places.
	got := ratetypes.MarginPercent(dec("1500"), dec("1600"))

	assert.True(t, got.Equal(dec("-6.6667")), "got %s", got)
}

// Margin is expressed as a share of revenue, and a shipment with no sell price
// has no revenue to take a share of. Dividing anyway would panic, and returning
// something made up would let a guardrail fire on a number that means nothing.
func TestMarginPercent_IsZeroWhenThereIsNoSellPrice(t *testing.T) {
	t.Parallel()

	assert.True(t, ratetypes.MarginPercent(decimal.Zero, dec("1600")).IsZero())
}

func TestMarginPercent_IsAHundredWhenTheLoadCostNothing(t *testing.T) {
	t.Parallel()

	assert.True(t, ratetypes.MarginPercent(dec("2000"), decimal.Zero).Equal(dec("100")))
}

// The floor is the number somebody negotiated, so a load landing exactly on it
// clears. Treating equality as a breach would block the deal the contract was
// written to allow.
func TestEvaluateMargin_ExactlyAtTheFloorClears(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:  dec("2000"),
		Buy:   dec("1700"),
		Floor: decimal.NewNullDecimal(dec("15")),
	})

	require.True(t, verdict.FloorApplies)
	assert.False(t, verdict.BelowFloor)
	assert.True(t, verdict.Percent.Equal(dec("15")), "percent was %s", verdict.Percent)
}

func TestEvaluateMargin_BelowTheFloorIsFlagged(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:  dec("2000"),
		Buy:   dec("1800"),
		Floor: decimal.NewNullDecimal(dec("15")),
	})

	require.True(t, verdict.FloorApplies)
	assert.True(t, verdict.BelowFloor)
	assert.Contains(t, verdict.Explanation, "10%")
	assert.Contains(t, verdict.Explanation, "15%")
}

// A contract that never named a floor has not agreed to one, and inventing a
// default would block loads on a number nobody negotiated.
func TestEvaluateMargin_NoFloorNeverBreaches(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell: dec("2000"),
		Buy:  dec("2500"),
	})

	assert.False(t, verdict.FloorApplies)
	assert.False(t, verdict.BelowFloor)
	assert.True(t, verdict.Amount.Equal(dec("-500")))
}

// The other half of the same guardrail, written from the carrier's side: a buy
// side contract can cap what it pays as a share of what the load sold for.
func TestEvaluateMargin_PayCeilingIsBreachedWhenThePayIsTooLargeAShare(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      dec("2000"),
		Buy:       dec("1800"),
		MaxPayPct: decimal.NewNullDecimal(dec("85")),
	})

	require.True(t, verdict.CeilingApplies)
	assert.True(t, verdict.AbovePayCeiling)
	assert.Contains(t, verdict.Explanation, "90%")
	assert.Contains(t, verdict.Explanation, "85%")
}

func TestEvaluateMargin_PayExactlyAtTheCeilingClears(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      dec("2000"),
		Buy:       dec("1700"),
		MaxPayPct: decimal.NewNullDecimal(dec("85")),
	})

	assert.False(t, verdict.AbovePayCeiling)
}

// Both guardrails can be breached at once, and the person reading it needs to
// be told about both rather than only whichever was checked first.
func TestEvaluateMargin_BothGuardrailsAreReportedTogether(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      dec("2000"),
		Buy:       dec("1900"),
		Floor:     decimal.NewNullDecimal(dec("15")),
		MaxPayPct: decimal.NewNullDecimal(dec("85")),
	})

	assert.True(t, verdict.BelowFloor)
	assert.True(t, verdict.AbovePayCeiling)
	assert.True(t, verdict.Breached())
}

// A shipment nobody has priced yet is not a margin breach. Rating the buy side
// before the sell side is ordinary — a broker quotes cost first — and firing a
// guardrail on it would block every one of those.
func TestEvaluateMargin_UnpricedSellSideIsNotABreach(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      decimal.Zero,
		Buy:       dec("1600"),
		Floor:     decimal.NewNullDecimal(dec("15")),
		MaxPayPct: decimal.NewNullDecimal(dec("85")),
	})

	assert.False(t, verdict.Breached())
	assert.False(t, verdict.FloorApplies)
	assert.False(t, verdict.CeilingApplies)
}

// A percentage the contract stated as zero is not the same as one it never
// stated. Zero means "never lose money on this", and it has to be enforceable.
func TestEvaluateMargin_AZeroFloorIsStillAFloor(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:  dec("2000"),
		Buy:   dec("2100"),
		Floor: decimal.NewNullDecimal(decimal.Zero),
	})

	require.True(t, verdict.FloorApplies)
	assert.True(t, verdict.BelowFloor)
}

func TestMarginVerdict_ExplanationIsEmptyWhenNothingIsBreached(t *testing.T) {
	t.Parallel()

	verdict := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:  dec("2000"),
		Buy:   dec("1000"),
		Floor: decimal.NewNullDecimal(dec("15")),
	})

	assert.Empty(t, verdict.Explanation)
	assert.False(t, verdict.Breached())
}
