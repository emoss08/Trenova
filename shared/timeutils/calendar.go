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
