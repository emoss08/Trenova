package accountsreceivableservice

import (
	"testing"

	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
)

func TestComputeDSODays(t *testing.T) {
	t.Run("zero billed returns zero", func(t *testing.T) {
		assert.Zero(t, computeDSODays(100_000, 0))
	})

	t.Run("balance equal to billed returns full window", func(t *testing.T) {
		assert.InDelta(t, 91.0, computeDSODays(500_000, 500_000), 0.001)
	})

	t.Run("half balance returns half window", func(t *testing.T) {
		assert.InDelta(t, 45.5, computeDSODays(250_000, 500_000), 0.001)
	})
}

func TestComputeCEI(t *testing.T) {
	t.Run("nil totals returns zero", func(t *testing.T) {
		assert.Zero(t, computeCEI(nil))
	})

	t.Run("everything collected returns 100", func(t *testing.T) {
		totals := &repositoryports.ARCollectionTotals{
			BeginningOpenMinor: 100_000,
			CreditSalesMinor:   200_000,
			EndingOpenMinor:    0,
			EndingCurrentMinor: 0,
		}
		assert.InDelta(t, 100.0, computeCEI(totals), 0.001)
	})

	t.Run("nothing collected returns zero", func(t *testing.T) {
		totals := &repositoryports.ARCollectionTotals{
			BeginningOpenMinor: 100_000,
			CreditSalesMinor:   200_000,
			EndingOpenMinor:    300_000,
			EndingCurrentMinor: 0,
		}
		assert.Zero(t, computeCEI(totals))
	})

	t.Run("partial collection between bounds", func(t *testing.T) {
		totals := &repositoryports.ARCollectionTotals{
			BeginningOpenMinor: 100_000,
			CreditSalesMinor:   100_000,
			EndingOpenMinor:    100_000,
			EndingCurrentMinor: 50_000,
		}
		assert.InDelta(t, 66.667, computeCEI(totals), 0.01)
	})

	t.Run("zero denominator returns zero", func(t *testing.T) {
		totals := &repositoryports.ARCollectionTotals{}
		assert.Zero(t, computeCEI(totals))
	})
}

func TestWorklistSeverity(t *testing.T) {
	assert.Equal(
		t,
		worklistSeverityCritical,
		worklistSeverity(&repositoryports.ARCollectionsWorklistItem{DaysPastDue: 30}),
	)
	assert.Equal(
		t,
		worklistSeverityWarning,
		worklistSeverity(&repositoryports.ARCollectionsWorklistItem{DaysPastDue: 15}),
	)
	assert.Equal(
		t,
		worklistSeverityWarning,
		worklistSeverity(&repositoryports.ARCollectionsWorklistItem{IsDisputed: true}),
	)
	assert.Equal(
		t,
		worklistSeverityWatch,
		worklistSeverity(&repositoryports.ARCollectionsWorklistItem{HasShortPay: true}),
	)
}

func TestComputeDelinquencyScore(t *testing.T) {
	t.Run("nil snapshot returns zero", func(t *testing.T) {
		assert.Zero(t, computeDelinquencyScore(nil))
	})

	t.Run("no open balance returns zero", func(t *testing.T) {
		assert.Zero(t, computeDelinquencyScore(&repositoryports.ARCustomerSnapshot{}))
	})

	t.Run("fully overdue aged customer maxes out", func(t *testing.T) {
		score := computeDelinquencyScore(&repositoryports.ARCustomerSnapshot{
			TotalOpenMinor:    100_000,
			OverdueMinor:      100_000,
			OldestDaysPastDue: 120,
			AvgDaysToPay:      120,
		})
		assert.InDelta(t, 100.0, score, 0.001)
	})

	t.Run("current customer scores low", func(t *testing.T) {
		score := computeDelinquencyScore(&repositoryports.ARCustomerSnapshot{
			TotalOpenMinor: 100_000,
			OverdueMinor:   0,
			AvgDaysToPay:   25,
		})
		assert.Zero(t, score)
	})
}

func TestClamps(t *testing.T) {
	assert.Equal(t, defaultTrendWeeks, clampWeeks(0))
	assert.Equal(t, maxTrendWeeks, clampWeeks(200))
	assert.Equal(t, 26, clampWeeks(26))
	assert.Equal(t, defaultTopOverdueLimit, clampLimit(0, defaultTopOverdueLimit))
	assert.Equal(t, maxListLimit, clampLimit(1000, defaultTopOverdueLimit))
}
