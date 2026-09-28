package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarUTC(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, time.September, 16, 23, 30, 0, 0, time.UTC).Unix()

	assert.Equal(t, "202609", MonthKeyUTC(ts))
	assert.Equal(t, 20260916, DayKeyUTC(ts))
	assert.Equal(t, "20260916", FormatDateKeyUTC(ts))
	assert.Equal(
		t,
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC).Unix(),
		MonthStartUTC(ts),
	)
	assert.Equal(
		t,
		time.Date(2026, time.September, 16, 0, 0, 0, 0, time.UTC).Unix(),
		DayStartUTC(ts),
	)
	assert.Equal(t, DayIndexUTC(ts), DayIndexUTC(DayStartUTC(ts)))
	assert.Equal(t, int64(3), WholeDaysBetween(ts, ts+3*SecondsPerDay+100))
	assert.Equal(t, int64(-1), WholeDaysBetween(ts, ts-SecondsPerDay))
}

func TestFormatInstantUTC(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "2026-09-24T17:00:00Z", FormatInstantUTC(1790269200))
	assert.Equal(t, "1970-01-01T00:00:00Z", FormatInstantUTC(0))
}

func TestFormatCalendarDateUsesTheZone(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 9, 25, 2, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t, "2026-09-25", FormatCalendarDate(ts, nil))
	assert.Equal(t, "2026-09-24", FormatCalendarDate(ts, LoadLocation("America/Chicago")))
}

func TestParseCalendarDateIsMidnightInTheLocation(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	got, err := ParseCalendarDate("2026-09-24", chicago)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, chicago).Unix(), got)
	assert.Equal(t, "2026-09-24", FormatCalendarDate(got, chicago))

	utc, err := ParseCalendarDate("2026-09-24", nil)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC).Unix(), utc)

	_, err = ParseCalendarDate("2026-02-30", time.UTC)
	require.Error(t, err)
}

func TestDayBoundariesFollowTheZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	evening := time.Date(2026, time.March, 7, 22, 30, 0, 0, chicago).Unix()

	assert.Equal(t, time.Date(2026, time.March, 7, 0, 0, 0, 0, chicago).Unix(), DayStart(evening, chicago))
	assert.Equal(t, time.Date(2026, time.March, 8, 0, 0, 0, 0, time.UTC).Unix(), DayStart(evening, nil))

	next := NextDayStart(evening, chicago)
	assert.Equal(t, time.Date(2026, time.March, 8, 0, 0, 0, 0, chicago).Unix(), next)
	afterChange := NextDayStart(next, chicago)
	assert.Equal(t, int64(23*3600), afterChange-next, "the day clocks spring forward is 23 hours long")

	assert.Equal(t, time.Date(2026, time.March, 6, 0, 0, 0, 0, chicago).Unix(), PreviousDayStart(evening, chicago))
	assert.Equal(t, time.Date(2026, time.February, 28, 0, 0, 0, 0, chicago).Unix(),
		PreviousDayStart(time.Date(2026, time.March, 1, 9, 0, 0, 0, chicago).Unix(), chicago))
}
