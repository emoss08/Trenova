package customerstatement

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthlyPeriodIsTheMonthBefore(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	now := time.Date(2026, time.September, 3, 9, 0, 0, 0, chicago).Unix()
	period, due := MonthlyPeriod(now, chicago, 1)

	assert.True(t, due)
	assert.Equal(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, chicago).Unix(), period.Start)
	assert.Equal(t, time.Date(2026, time.August, 31, 23, 59, 59, 0, chicago).Unix(), period.End)
}

func TestMonthlyPeriodWaitsForTheDay(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC).Unix()

	_, due := MonthlyPeriod(now, time.UTC, 5)
	assert.False(t, due)

	_, due = MonthlyPeriod(now+24*60*60, time.UTC, 5)
	assert.True(t, due)
}

func TestMonthlyPeriodUsesTheLocalDate(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 20:00 UTC on 31 August is already 1 September in Tokyo.
	now := time.Date(2026, time.August, 31, 20, 0, 0, 0, time.UTC).Unix()
	period, due := MonthlyPeriod(now, tokyo, 1)

	assert.True(t, due)
	assert.Equal(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, tokyo).Unix(), period.Start)
}

func TestMonthlyPeriodAcrossTheYear(t *testing.T) {
	t.Parallel()

	now := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	period, due := MonthlyPeriod(now, nil, 40)

	assert.True(t, due, "a day out of range is treated as the first")
	assert.Equal(t, time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC).Unix(), period.Start)
	assert.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()-1, period.End)
}
