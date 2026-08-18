package ratesimulation_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func summarize(deltas ...ratesimulation.Delta) ratesimulation.Summary {
	var acc ratesimulation.Accumulator

	for _, delta := range deltas {
		acc.Add(delta)
	}

	return acc.Summary()
}

func priced(before, after string) ratesimulation.Delta {
	return ratesimulation.Delta{Before: dec(before), After: dec(after)}
}

func TestSummary_CountsEveryShipmentItSaw(t *testing.T) {
	t.Parallel()

	summary := summarize(priced("100", "110"), priced("200", "200"))

	assert.Equal(t, 2, summary.ShipmentCount)
	assert.Equal(t, 2, summary.EvaluatedCount)
}

// A shipment priced the same both ways is not a change, and counting it as one
// would make every simulation look like it moved everything.
func TestSummary_AShipmentPricedTheSameIsNotAChange(t *testing.T) {
	t.Parallel()

	summary := summarize(priced("200", "200"))

	assert.Equal(t, 0, summary.ChangedCount)
	assert.Equal(t, 1, summary.EvaluatedCount)
}

func TestSummary_SplitsIncreasesFromDecreases(t *testing.T) {
	t.Parallel()

	summary := summarize(priced("100", "150"), priced("100", "80"), priced("100", "100"))

	assert.Equal(t, 2, summary.ChangedCount)
	assert.Equal(t, 1, summary.IncreasedCount)
	assert.Equal(t, 1, summary.DecreasedCount)
}

// A shipment that could not be priced either way contributes nothing to the
// totals. Folding a zero into them would understate the revenue a simulation is
// being judged on.
func TestSummary_AFailedShipmentIsCountedButNotTotalled(t *testing.T) {
	t.Parallel()

	summary := summarize(
		priced("100", "150"),
		ratesimulation.Delta{Failed: true, Before: dec("999"), After: dec("999")},
	)

	assert.Equal(t, 2, summary.ShipmentCount)
	assert.Equal(t, 1, summary.EvaluatedCount)
	assert.Equal(t, 1, summary.ErrorCount)
	assert.True(t, summary.BeforeTotal.Equal(dec("100")), "before total %s", summary.BeforeTotal)
	assert.True(t, summary.AfterTotal.Equal(dec("150")), "after total %s", summary.AfterTotal)
}

func TestSummary_TotalDeltaIsTheDifferenceOfTheTotals(t *testing.T) {
	t.Parallel()

	summary := summarize(priced("100", "150"), priced("200", "180"))

	assert.True(t, summary.TotalDelta.Equal(dec("30")), "total delta %s", summary.TotalDelta)
	// 30 of 300 is 10%.
	assert.True(t, summary.TotalDeltaPct.Equal(dec("10")), "pct %s", summary.TotalDeltaPct)
}

// A simulation against shipments that were all priced at nothing has no base to
// take a percentage of, and dividing anyway would panic.
func TestSummary_PercentIsZeroWhenNothingWasPricedBefore(t *testing.T) {
	t.Parallel()

	summary := summarize(priced("0", "150"))

	assert.True(t, summary.TotalDeltaPct.IsZero())
	assert.True(t, summary.TotalDelta.Equal(dec("150")))
}

// The largest single move in each direction is what somebody scans for: it is
// the shipment that will produce the phone call.
func TestSummary_RemembersTheLargestMoveInEachDirection(t *testing.T) {
	t.Parallel()

	summary := summarize(
		priced("100", "140"),
		priced("100", "175"),
		priced("100", "60"),
		priced("100", "95"),
	)

	assert.True(t, summary.MaxIncrease.Equal(dec("75")), "max increase %s", summary.MaxIncrease)
	assert.True(t, summary.MaxDecrease.Equal(dec("-40")), "max decrease %s", summary.MaxDecrease)
}

func TestSummary_AnEmptyRunSummarizesToNothing(t *testing.T) {
	t.Parallel()

	summary := summarize()

	assert.Zero(t, summary.ShipmentCount)
	assert.True(t, summary.TotalDelta.IsZero())
	assert.True(t, summary.TotalDeltaPct.IsZero())
}

// A simulation walks a year of shipments, which is far too many to hold in
// memory at once. The accumulator has to give the same answer fed one at a time
// as it would from a slice.
func TestSummary_AccumulatesWithoutHoldingTheShipments(t *testing.T) {
	t.Parallel()

	var acc ratesimulation.Accumulator

	for i := range 1000 {
		acc.Add(priced("100", "110"))
		_ = i
	}

	summary := acc.Summary()

	require.Equal(t, 1000, summary.EvaluatedCount)
	assert.True(t, summary.TotalDelta.Equal(dec("10000")), "total delta %s", summary.TotalDelta)
	assert.True(t, summary.TotalDeltaPct.Equal(dec("10")), "pct %s", summary.TotalDeltaPct)
}

// Percent per shipment answers "how much did this one move", which is the
// column somebody sorts by to find the outliers.
func TestDelta_PercentIsTheMoveAgainstWhatItWasPricedAt(t *testing.T) {
	t.Parallel()

	delta := priced("200", "250")

	assert.True(t, delta.Amount().Equal(dec("50")))
	assert.True(t, delta.Percent().Equal(dec("25")), "percent %s", delta.Percent())
}

func TestDelta_PercentIsZeroAgainstAShipmentPricedAtNothing(t *testing.T) {
	t.Parallel()

	assert.True(t, priced("0", "250").Percent().IsZero())
}
