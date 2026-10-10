package worker

import (
	"time"

	"github.com/shopspring/decimal"
)

func localDate(unix int64, loc *time.Location) time.Time {
	t := time.Unix(unix, 0).In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// ComputePTODays counts the inclusive calendar days a request covers in the
// organisation's local calendar. When the policy does not count weekends,
// Saturdays, Sundays and observed holidays are skipped; a policy that counts
// weekends counts every day. A nil calendar simply has no holidays.
func ComputePTODays(
	start, end int64,
	loc *time.Location,
	countWeekends bool,
	holidays *HolidayCalendar,
) decimal.Decimal {
	if loc == nil {
		loc = time.UTC
	}
	first := localDate(start, loc)
	last := localDate(end, loc)
	if last.Before(first) {
		return decimal.Zero
	}

	var days int64
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if countWeekends {
			days++
			continue
		}
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		if holidays.HolidayOn(d) {
			continue
		}
		days++
	}

	return decimal.NewFromInt(days)
}
