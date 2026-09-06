package timeutils

import "time"

// AddMonthsUTC moves a Unix timestamp forward by whole calendar months in UTC,
// clamping to the last day of a shorter month so 31 January + 1 month is
// 28 February, never 3 March. The time of day is preserved.
func AddMonthsUTC(ts int64, months int) int64 {
	if months == 0 {
		return ts
	}
	t := time.Unix(ts, 0).UTC()
	year, month, day := t.Date()
	target := time.Date(year, month+time.Month(months), 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	lastDay := time.Date(target.Year(), target.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(target.Year(), target.Month(), day, t.Hour(), t.Minute(), t.Second(), 0, time.UTC).Unix()
}

// YearOfUnix is the UTC calendar year an instant falls in. Regulatory logs are
// kept by calendar year regardless of anybody's fiscal year or local time, so
// this deliberately does not take a timezone.
func YearOfUnix(ts int64) int {
	return time.Unix(ts, 0).UTC().Year()
}
