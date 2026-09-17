package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
