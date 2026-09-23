package timeutils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The span of epoch seconds a record could plausibly be reporting as a date.
// The bound is what separates a timestamp from a large quantity that happens to
// share a date-like name: an invoice total in cents can land in this range, a
// due date cannot land outside it.
const (
	earliestPlausibleInstant int64 = 946684800  // 2000-01-01
	latestPlausibleInstant   int64 = 4102444800 // 2100-01-01
)

// IsPlausibleInstant reports whether seconds could be a date a record holds.
// Zero, which the row types use for "not set", never is.
func IsPlausibleInstant(seconds int64) bool {
	return seconds >= earliestPlausibleInstant && seconds <= latestPlausibleInstant
}

func NowUnix() int64 {
	return time.Now().Unix()
}

func CeilSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	secs := int(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	return secs
}

func WithDefaultDuration(val, def time.Duration) time.Duration {
	if val == 0 {
		return def
	}

	return val
}

func ParseTimeRFC3339(value string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, true
	}
	if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		if timestamp > 1e12 {
			return time.Unix(timestamp/1000, (timestamp%1000)*1e6), true
		}
		return time.Unix(timestamp, 0), true
	}
	return time.Time{}, false
}

func UnixToHumanReadable(ts int64) string {
	return time.Unix(ts, 0).Format("January 2, 2006 3:04:05 PM MST")
}

// FormatUnixDate renders a unix timestamp as a calendar date without a time component,
// for messages where the clock time carries no meaning (time off, due dates).
func FormatUnixDate(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("Jan 2, 2006")
}

func NormalizeTimezone(timezone string) string {
	if timezone == "" {
		return "UTC"
	}

	return timezone
}

func NowAddDuration(duration time.Duration) int64 {
	return time.Now().Add(duration).Unix()
}

func DayStartUnix(ts int64, timezone string) (int64, error) {
	if ts <= 0 {
		return 0, fmt.Errorf("timestamp must be greater than zero")
	}

	loc, err := time.LoadLocation(NormalizeTimezone(timezone))
	if err != nil {
		return 0, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	t := time.Unix(ts, 0).In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start.Unix(), nil
}

func DayEndUnix(ts int64, timezone string) (int64, error) {
	if ts <= 0 {
		return 0, fmt.Errorf("timestamp must be greater than zero")
	}

	loc, err := time.LoadLocation(NormalizeTimezone(timezone))
	if err != nil {
		return 0, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	t := time.Unix(ts, 0).In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1).Add(-time.Second)
	return end.Unix(), nil
}

func CurrentDateInTimezone(location string) string {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func TimeZoneAwareNow(location string) int64 {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return NowUnix()
	}
	now := time.Now().In(loc)
	return now.Unix()
}

// DescribeUnixDate renders an instant as a calendar date and how far away it is.
//
// It exists because a bare epoch integer is unreadable to anything that cannot
// do arithmetic, and a language model cannot. Asked which medical cards expired
// within thirty days, one was handed 1791591001, estimated "56 years × 365.25
// days", called it July 2026 when it was October 2026, and told a dispatcher
// that nobody was expiring when somebody was expiring in three weeks.
//
// The ISO prefix is deliberate: it is unambiguous, it sorts, and it is the same
// form the date filters accept, so a value read out of one result can be passed
// straight back into the next call.
func DescribeUnixDate(ts, now int64) string {
	return DescribeUnixDateIn(ts, now, "UTC")
}

// DescribeUnixDateIn is DescribeUnixDate with the day boundary drawn in the
// given zone. Which day an instant falls on is a question about a place: a
// card expiring at 00:30 New York time is "tomorrow" to the dispatcher there
// at 23:30, and "today" only to a clock in Greenwich. An empty or unknown zone
// is UTC.
func DescribeUnixDateIn(ts, now int64, timezone string) string {
	loc := LoadLocation(timezone)
	date := time.Unix(ts, 0).In(loc).Format(time.DateOnly)

	startOfDay := func(seconds int64) time.Time {
		t := time.Unix(seconds, 0).In(loc)

		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}

	// Whole days apart, not seconds apart. "Tomorrow" has to read as one day
	// away at any hour, or a card expiring tonight and one expiring tomorrow
	// morning differ by a rounding decision nobody can see.
	days := int64(startOfDay(ts).Sub(startOfDay(now)).Hours() / 24)

	switch {
	case days == 0:
		return date + " (today)"
	case days == 1:
		return date + " (tomorrow)"
	case days == -1:
		return date + " (yesterday)"
	case days > 1:
		return fmt.Sprintf("%s (in %d days)", date, days)
	default:
		return fmt.Sprintf("%s (%d days ago)", date, -days)
	}
}

// DescribeUnixInstantIn is DescribeUnixDateIn with the time of day, for an
// instant whose hour matters: an appointment window, an arrival, a departure.
// "2026-09-24 14:00 CDT (in 2 days)" keeps the date first, so the value still
// reads back into a date filter.
func DescribeUnixInstantIn(ts, now int64, timezone string) string {
	clock := time.Unix(ts, 0).In(LoadLocation(timezone)).Format("15:04 MST")
	date, distance, _ := strings.Cut(DescribeUnixDateIn(ts, now, timezone), " ")

	return date + " " + clock + " " + distance
}
