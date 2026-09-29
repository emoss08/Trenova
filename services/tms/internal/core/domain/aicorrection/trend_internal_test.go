package aicorrection

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var trendNow = time.Date(2026, time.September, 30, 15, 0, 0, 0, time.UTC).Unix()

func weekBefore(window TrendWindow, weeks int) int64 {
	return window.CheckedWeek - int64(weeks)*secondsPerWeek
}

func TestTrendWindowEndsWithTheCurrentWeek(t *testing.T) {
	t.Parallel()

	window := NewTrendWindow(trendNow)
	monday := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC).Unix()

	require.Len(t, window.Weeks, TrendWeeks)
	assert.Equal(t, monday, window.Weeks[TrendWeeks-1])
	assert.Equal(t, monday-secondsPerWeek, window.CheckedWeek)
	assert.Equal(t, window.CheckedWeek-DriftBaselineWeeks*secondsPerWeek, window.BaselineStart)
	assert.Equal(t, window.Weeks[0], window.Since())
	for i := 1; i < len(window.Weeks); i++ {
		assert.Equal(t, int64(secondsPerWeek), window.Weeks[i]-window.Weeks[i-1])
	}
}

func TestProviderTrendFlagsADropAgainstItsOwnBaseline(t *testing.T) {
	t.Parallel()

	window := NewTrendWindow(trendNow)
	provider := pulid.MustNew("aip_")
	totals := []WeekTotal{
		{ProviderID: provider, WeekStart: window.CheckedWeek, Corrections: 6, Scored: 120, Correct: 96},
		{ProviderID: provider, WeekStart: window.Weeks[TrendWeeks-1], Corrections: 1, Scored: 20, Correct: 5},
		{ProviderID: pulid.Nil, WeekStart: window.CheckedWeek, Scored: 500, Correct: 1},
		{ProviderID: provider, WeekStart: window.CheckedWeek - 100*secondsPerWeek, Scored: 500, Correct: 0},
	}
	for weeks := 1; weeks <= DriftBaselineWeeks; weeks++ {
		totals = append(totals, WeekTotal{
			ProviderID:  provider,
			WeekStart:   weekBefore(window, weeks),
			Corrections: 5,
			Scored:      100,
			Correct:     92,
		})
	}

	trends := BuildProviderTrends(window, totals)

	require.Len(t, trends, 1)
	trend := trends[0]
	assert.Equal(t, provider, trend.ProviderID)
	assert.InDelta(t, 0.8, trend.Checked.Accuracy, 0.0001)
	assert.Equal(t, 400, trend.Baseline.Scored)
	assert.InDelta(t, 0.92, trend.Baseline.Accuracy, 0.0001)
	assert.True(t, trend.Comparable)
	assert.InDelta(t, 12, trend.DropPoints, 0.0001)
	assert.True(t, trend.Drifting)
	assert.Equal(t, 20, trend.Weeks[TrendWeeks-1].Scored, "the current week is shown but not judged")
	assert.Zero(t, trend.Weeks[0].Scored, "a week without corrections is an empty point")
}

func TestProviderTrendNeedsEnoughEvidence(t *testing.T) {
	t.Parallel()

	window := NewTrendWindow(trendNow)
	provider := pulid.MustNew("aip_")
	totals := []WeekTotal{
		{ProviderID: provider, WeekStart: window.CheckedWeek, Scored: MinDriftWeekFields - 1, Correct: 0},
		{ProviderID: provider, WeekStart: weekBefore(window, 1), Scored: MinDriftBaselineFields, Correct: MinDriftBaselineFields},
	}

	trend := BuildProviderTrends(window, totals)[0]
	assert.False(t, trend.Comparable)
	assert.False(t, trend.Drifting)
	assert.Zero(t, trend.DropPoints)
}

func TestProviderTrendAllowsTheDriftPoints(t *testing.T) {
	t.Parallel()

	window := NewTrendWindow(trendNow)
	provider := pulid.MustNew("aip_")
	totals := []WeekTotal{
		{ProviderID: provider, WeekStart: window.CheckedWeek, Scored: 100, Correct: 85},
		{ProviderID: provider, WeekStart: weekBefore(window, 2), Scored: 200, Correct: 180},
	}

	trend := BuildProviderTrends(window, totals)[0]
	assert.True(t, trend.Comparable)
	assert.InDelta(t, DriftPoints, trend.DropPoints, 0.0001)
	assert.False(t, trend.Drifting, "exactly the allowed drop is not drift")

	improving := BuildProviderTrends(window, []WeekTotal{
		{ProviderID: provider, WeekStart: window.CheckedWeek, Scored: 100, Correct: 99},
		{ProviderID: provider, WeekStart: weekBefore(window, 1), Scored: 200, Correct: 150},
	})[0]
	assert.Negative(t, improving.DropPoints)
	assert.False(t, improving.Drifting)
}

func TestProviderTrendsPutTheBusiestFirst(t *testing.T) {
	t.Parallel()

	window := NewTrendWindow(trendNow)
	quiet := pulid.MustNew("aip_")
	busy := pulid.MustNew("aip_")
	trends := BuildProviderTrends(window, []WeekTotal{
		{ProviderID: quiet, WeekStart: window.CheckedWeek, Scored: 10, Correct: 9},
		{ProviderID: busy, WeekStart: window.CheckedWeek, Scored: 300, Correct: 280},
	})

	require.Len(t, trends, 2)
	assert.Equal(t, busy, trends[0].ProviderID)
	assert.Equal(t, quiet, trends[1].ProviderID)
}
