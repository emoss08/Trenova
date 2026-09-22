package filtercatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 2026-09-20T03:30:00Z: 23:30 on the 19th in New York.
const lateEveningNewYork int64 = 1789875000

func TestClock_TodayIsMidnightInTheZone(t *testing.T) {
	t.Parallel()

	utc := Clock{Now: lateEveningNewYork}
	assert.Equal(t, int64(1789862400), utc.Today(), "UTC: the 20th")

	ny := Clock{Now: lateEveningNewYork, Timezone: "America/New_York"}
	assert.Equal(t, int64(1789790400), ny.Today(), "New York: the 19th, at 04:00Z")
}

// A day across a Clock change is 23 or 25 hours long; the count must still be
// whole days. The US fall-back is the first Sunday of November.
func TestClock_DaysBetweenCountsCalendarDaysAcrossAClockChange(t *testing.T) {
	t.Parallel()

	ny := Clock{Timezone: "America/New_York"}
	const oct31 int64 = 1793419200 // 2026-10-31T00:00:00-04:00
	const nov3 int64 = 1793682000  // 2026-11-03T00:00:00-05:00, 73 hours later

	assert.Equal(t, int64(3), ny.DaysBetween(oct31, nov3))
	assert.Equal(t, int64(-3), ny.DaysBetween(nov3, oct31))
}

func TestClock_ParsesACalendarDateAsMidnightInTheZone(t *testing.T) {
	t.Parallel()

	ny := Clock{Timezone: "America/New_York"}
	seconds, ok := ny.ParseDate("2026-09-19")
	assert.True(t, ok)
	assert.Equal(t, int64(1789790400), seconds, "04:00Z")

	seconds, ok = ny.ParseDate("2026-09-19T12:00:00Z")
	assert.True(t, ok)
	assert.Equal(t, int64(1789819200), seconds, "an instant is itself")

	_, ok = ny.ParseDate("next tuesday")
	assert.False(t, ok)
}

// A zero Clock is what a criteria built without one carries. It reads the
// wall Clock rather than the epoch, so nothing downstream has to know.
func TestClock_ZeroValueReadsTheWallClock(t *testing.T) {
	t.Parallel()

	var zero Clock
	assert.Greater(t, zero.Instant(), int64(1789875000))
	assert.Equal(t, "UTC", zero.Location().String())
}
