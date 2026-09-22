package filtercatalog

import (
	"math"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
)

// SecondsPerDay is a day as the filters count one.
const SecondsPerDay = 86400

// Clock is the frame a query reads dates in: this instant, and the zone whose
// days are meant.
//
// It exists because "the next 30 days" is not a number of seconds. A window
// that opened at the current second excluded a medical card that expired at
// nine this morning — the one question the compliance tools were built to
// answer — and one that opened at UTC midnight was right only for organizations
// in Greenwich. The zone is the organization's; callers receive it on every
// request and never guess.
type Clock struct {
	Now      int64
	Timezone string
}

// NewClock reads this instant in the given zone.
func NewClock(timezone string) Clock {
	return Clock{Now: timeutils.NowUnix(), Timezone: timezone}
}

// Instant is now. A zero Clock, which is what a Criteria built without one
// carries, reads the wall clock so nothing downstream has to special-case it.
func (c Clock) Instant() int64 {
	if c.Now > 0 {
		return c.Now
	}

	return timeutils.NowUnix()
}

func (c Clock) Location() *time.Location {
	return timeutils.LoadLocation(c.Timezone)
}

// DayStart is midnight of the day the instant falls on, in the zone.
func (c Clock) DayStart(ts int64) int64 {
	start, err := timeutils.DayStartUnix(ts, c.Timezone)
	if err != nil {
		return ts - ts%SecondsPerDay
	}

	return start
}

func (c Clock) Today() int64 {
	return c.DayStart(c.Instant())
}

// DaysBetween counts calendar days from one instant to another, in the zone.
// A day across a clock change is 23 or 25 hours long, so the count is rounded
// rather than truncated: the difference between two midnights is never off by
// an hour, but it is often not a multiple of 86400.
func (c Clock) DaysBetween(from, to int64) int64 {
	diff := float64(c.DayStart(to) - c.DayStart(from))

	return int64(math.Round(diff / SecondsPerDay))
}

// ParseDate reads a calendar date as midnight of that day in the zone, or an
// RFC 3339 instant as itself.
func (c Clock) ParseDate(raw string) (int64, bool) {
	if parsed, err := time.ParseInLocation(time.DateOnly, raw, c.Location()); err == nil {
		return parsed.Unix(), true
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Unix(), true
	}

	return 0, false
}
