package agentquerytoolservice

import (
	"math"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
)

// clock is the frame a query reads dates in: this instant, and the zone whose
// days are meant.
//
// It exists because "the next 30 days" is not a number of seconds. A window
// that opened at the current second excluded a medical card that expired at
// nine this morning — the one question the compliance tools were built to
// answer — and one that opened at UTC midnight was right only for
// organizations in Greenwich. The zone is the organization's; the tools
// receive it on every call and never guess.
type clock struct {
	now      int64
	timezone string
}

func clockFor(params serviceports.QueryToolParams) clock {
	return clock{now: timeutils.NowUnix(), timezone: params.Timezone}
}

// instant is now. A zero clock, which is what a criteria built without one
// carries, reads the wall clock so nothing downstream has to special-case it.
func (c clock) instant() int64 {
	if c.now > 0 {
		return c.now
	}

	return timeutils.NowUnix()
}

func (c clock) location() *time.Location {
	return timeutils.LoadLocation(c.timezone)
}

// dayStart is midnight of the day the instant falls on, in the zone.
func (c clock) dayStart(ts int64) int64 {
	start, err := timeutils.DayStartUnix(ts, c.timezone)
	if err != nil {
		return ts - ts%secondsPerDay
	}

	return start
}

func (c clock) today() int64 {
	return c.dayStart(c.instant())
}

// daysBetween counts calendar days from one instant to another, in the zone.
// A day across a clock change is 23 or 25 hours long, so the count is rounded
// rather than truncated: the difference between two midnights is never off
// by an hour, but it is often not a multiple of 86400.
func (c clock) daysBetween(from, to int64) int64 {
	diff := float64(c.dayStart(to) - c.dayStart(from))

	return int64(math.Round(diff / secondsPerDay))
}

// parseDate reads a calendar date as midnight of that day in the zone, or an
// RFC 3339 instant as itself.
func (c clock) parseDate(raw string) (int64, bool) {
	if parsed, err := time.ParseInLocation(time.DateOnly, raw, c.location()); err == nil {
		return parsed.Unix(), true
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Unix(), true
	}

	return 0, false
}
