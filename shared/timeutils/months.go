package timeutils

import "time"

// AddMonthsUTC moves a Unix timestamp forward by whole calendar months in UTC,
// clamping to the last day of a shorter month so 31 January + 1 month is
// 28 February, never 3 March. The time of day is preserved.
func AddMonthsUTC(ts int64, months int) int64 {
	return AddMonths(time.Unix(ts, 0).UTC(), months).Unix()
}

// AddMonths moves t by whole calendar months in t's own location, clamping to
// the last day of a shorter month so 31 March - 1 month is 28 (or 29)
// February, never 3 March as time.AddDate normalizes it. The wall-clock time
// of day is preserved.
func AddMonths(t time.Time, months int) time.Time {
	if months == 0 {
		return t
	}
	year, month, day := t.Date()
	loc := t.Location()
	targetYear, targetMonth, _ := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, loc).
		Date()
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, loc).Day()
	day = min(day, lastDay)
	return time.Date(
		targetYear,
		targetMonth,
		day,
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		loc,
	)
}

// YearOfUnix is the UTC calendar year an instant falls in. Regulatory logs are
// kept by calendar year regardless of anybody's fiscal year or local time, so
// this deliberately does not take a timezone.
func YearOfUnix(ts int64) int {
	return time.Unix(ts, 0).UTC().Year()
}
