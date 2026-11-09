package timeutils

import "time"

const daysPerWeek = 7

func BusinessDaysBetween(from, to int64, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	if to <= from {
		return 0
	}
	start := civilDay(time.Unix(from, 0).In(loc))
	end := civilDay(time.Unix(to, 0).In(loc))
	span := int(end.Sub(start).Hours() / 24)
	if span <= 0 {
		return 0
	}

	days := (span / daysPerWeek) * 5
	day := start.AddDate(0, 0, (span/daysPerWeek)*daysPerWeek)
	for range span % daysPerWeek {
		day = day.AddDate(0, 0, 1)
		if weekday := day.Weekday(); weekday != time.Saturday && weekday != time.Sunday {
			days++
		}
	}

	return days
}

func civilDay(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}
