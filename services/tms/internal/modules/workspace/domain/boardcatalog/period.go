package boardcatalog

import "time"

const (
	daysPerWeek      = 7
	daysAhead        = 30
	monthsPerQuarter = 3
)

func (p Period) Days(now int64, loc *time.Location) (first, last time.Time, ok bool) {
	if loc == nil {
		loc = time.UTC
	}
	at := time.Unix(now, 0).In(loc)
	today := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + daysPerWeek - 1) % daysPerWeek))
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc)
	quarter := time.Date(
		today.Year(), today.Month()-(today.Month()-1)%monthsPerQuarter, 1, 0, 0, 0, 0, loc,
	)

	switch p {
	case PeriodToday:
		return today, today, true
	case PeriodYesterday:
		return today.AddDate(0, 0, -1), today.AddDate(0, 0, -1), true
	case PeriodThisWeek:
		return monday, today, true
	case PeriodLastWeek:
		return monday.AddDate(0, 0, -daysPerWeek), monday.AddDate(0, 0, -1), true
	case PeriodNextWeek:
		return monday.AddDate(0, 0, daysPerWeek), monday.AddDate(0, 0, 2*daysPerWeek-1), true
	case PeriodThisMonth:
		return month, today, true
	case PeriodLastMonth:
		return month.AddDate(0, -1, 0), month.AddDate(0, 0, -1), true
	case PeriodThisQuarter:
		return quarter, today, true
	case PeriodLastQuarter:
		return quarter.AddDate(0, -monthsPerQuarter, 0), quarter.AddDate(0, 0, -1), true
	case PeriodNext30Days:
		return today, today.AddDate(0, 0, daysAhead), true
	case PeriodNext48Hours, PeriodSlip:
	}

	return time.Time{}, time.Time{}, false
}
