package timeutils

import (
	"fmt"
	"time"
)

// The layouts here are shared by documents and messages so a date reads the same
// on an invoice PDF, in the email that delivers it, and in the notice that
// preceded it. A customer comparing the three should never have to reconcile
// three different renderings of the same instant.
const (
	// ISODateLayout is the unambiguous calendar date for anything a customer or
	// an auditor may have to match against a record in another system.
	ISODateLayout = "2006-01-02"

	// StampLayout names the day of week and the zone, because "arrived at 14:05"
	// is worth disputing and "arrived Mon 02 Jun 2026 14:05 CDT" is not.
	StampLayout = "Mon 02 Jan 2006 15:04 MST"

	// DateTimeLayout is the general-purpose timestamp for document headers.
	DateTimeLayout = "Jan 2, 2006 3:04 PM MST"
)

// LoadLocation resolves a timezone name, falling back to UTC.
//
// It never fails: a bad timezone on an organization record must not stop an
// invoice from rendering, and UTC is the honest fallback because it is the zone
// the timestamps are stored in.
func LoadLocation(timezone string) *time.Location {
	if timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// FormatUnixDateIn renders a calendar date in the given timezone.
//
// The zone matters more than it looks: a shipment delivered at 19:00 Pacific is
// the next day in UTC, and an invoice dated a day late is an invoice a customer
// can refuse.
func FormatUnixDateIn(ts int64, timezone string) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).In(LoadLocation(timezone)).Format(ISODateLayout)
}

// FormatUnixDateTimeIn renders a timestamp with its clock time and zone.
func FormatUnixDateTimeIn(ts int64, timezone string) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).In(LoadLocation(timezone)).Format(DateTimeLayout)
}

// FormatStamp renders a timestamp with the weekday and zone spelled out, for
// the arrival and departure times that detention charges are argued over.
func FormatStamp(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(ts, 0).In(loc).Format(StampLayout)
}

// FormatStampIn is FormatStamp keyed by timezone name.
func FormatStampIn(ts int64, timezone string) string {
	if ts == 0 {
		return ""
	}
	return FormatStamp(ts, LoadLocation(timezone))
}

// minutesPerHour is the divisor for the duration renderers below.
const minutesPerHour = 60

// FormatMinutes renders a whole number of minutes the way a dispatcher says it
// out loud: "45 min", "3h", "3h 20m".
func FormatMinutes(minutes int32) string {
	if minutes <= 0 {
		return "0 min"
	}
	if minutes < minutesPerHour {
		return fmt.Sprintf("%d min", minutes)
	}

	hours := minutes / minutesPerHour
	rem := minutes % minutesPerHour
	if rem == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, rem)
}
