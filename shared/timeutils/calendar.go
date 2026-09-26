package timeutils

import "time"

func MonthKeyUTC(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("200601")
}

func DayKeyUTC(ts int64) int {
	year, month, day := time.Unix(ts, 0).UTC().Date()
	return year*10000 + int(month)*100 + day
}

func MonthStartUTC(ts int64) int64 {
	year, month, _ := time.Unix(ts, 0).UTC().Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Unix()
}

func DayStartUTC(ts int64) int64 {
	year, month, day := time.Unix(ts, 0).UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func DayIndexUTC(ts int64) int64 {
	return DayStartUTC(ts) / SecondsPerDay
}

func WholeDaysBetween(from, to int64) int64 {
	return (to - from) / SecondsPerDay
}

func FormatDateKeyUTC(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("20060102")
}

// IsCalendarDate reports whether a string is a real YYYY-MM-DD day. It
// parses rather than pattern-matches, so 2026-02-30 is rejected the same
// way a malformed string is.
func IsCalendarDate(value string) bool {
	_, err := time.Parse(ISODateLayout, value)

	return err == nil
}

func FormatInstantUTC(ts int64) string {
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func FormatCalendarDate(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(ts, 0).In(loc).Format(ISODateLayout)
}

func ParseCalendarDate(value string, loc *time.Location) (int64, error) {
	if loc == nil {
		loc = time.UTC
	}
	day, err := time.ParseInLocation(ISODateLayout, value, loc)
	if err != nil {
		return 0, err
	}
	return day.Unix(), nil
}
