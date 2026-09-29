package timeutils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type WrittenDate struct {
	Day   CalendarDay
	Start int
	End   int
	Clock string
}

var (
	writtenISOPattern = regexp.MustCompile(
		`\b(\d{4})-(\d{1,2})-(\d{1,2})(?:[T ](\d{1,2}:\d{2})(?::\d{2})?)?\b`,
	)
	writtenNumericPattern = regexp.MustCompile(`\b(\d{1,2})([/-])(\d{1,2})([/-])(\d{4})\b`)
	writtenNamedPattern   = regexp.MustCompile(
		`(?i)\b(` + writtenMonthNames + `)\.?\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?\b`,
	)
	writtenDayFirstPattern = regexp.MustCompile(
		`(?i)\b(\d{1,2})(?:st|nd|rd|th)?\s+(` + writtenMonthNames + `)\.?(?:,?\s+(\d{4}))?\b`,
	)
)

const writtenMonthNames = `jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|` +
	`aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?`

func RelativeDays(days int64) string {
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case days == -1:
		return "yesterday"
	case days > 1:
		return fmt.Sprintf("in %d days", days)
	default:
		return fmt.Sprintf("%d days ago", -days)
	}
}

func CalendarDaysBetween(from, to time.Time) int64 {
	loc := from.Location()
	start := func(t time.Time) time.Time {
		local := t.In(loc)

		return time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, time.UTC)
	}

	return int64(start(to).Sub(start(from)).Hours() / 24)
}

func (d CalendarDay) On(now time.Time) time.Time {
	loc := now.Location()
	if d.HasYear() {
		return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
	}

	best := time.Date(now.Year(), d.Month, d.Day, 0, 0, 0, 0, loc)
	for _, year := range []int{now.Year() - 1, now.Year() + 1} {
		candidate := time.Date(year, d.Month, d.Day, 0, 0, 0, 0, loc)
		if candidate.Month() != d.Month {
			continue
		}
		if absDuration(candidate.Sub(now)) < absDuration(best.Sub(now)) {
			best = candidate
		}
	}

	return best
}

func FindWrittenDate(value string) (WrittenDate, bool) {
	if strings.TrimSpace(value) == "" {
		return WrittenDate{}, false
	}

	if match := writtenISOPattern.FindStringSubmatchIndex(value); match != nil {
		day, ok := newCalendarDay(
			atoi(value[match[2]:match[3]]),
			atoi(value[match[4]:match[5]]),
			atoi(value[match[6]:match[7]]),
		)
		if !ok {
			return WrittenDate{}, false
		}
		written := WrittenDate{Day: day, Start: match[0], End: match[1]}
		if match[8] >= 0 {
			written.Clock = value[match[8]:match[9]]
		}

		return written, true
	}

	if match := writtenNumericPattern.FindStringSubmatchIndex(value); match != nil &&
		value[match[4]:match[5]] == value[match[8]:match[9]] {
		day, ok := newCalendarDay(
			atoi(value[match[10]:match[11]]),
			atoi(value[match[2]:match[3]]),
			atoi(value[match[6]:match[7]]),
		)
		if !ok {
			return WrittenDate{}, false
		}

		return WrittenDate{Day: day, Start: match[0], End: match[1]}, true
	}

	if match := writtenNamedPattern.FindStringSubmatchIndex(value); match != nil {
		return namedWrittenDate(value, match, 2, 4)
	}

	if match := writtenDayFirstPattern.FindStringSubmatchIndex(value); match != nil {
		return namedWrittenDate(value, match, 4, 2)
	}

	return WrittenDate{}, false
}

func namedWrittenDate(value string, match []int, monthGroup, dayGroup int) (WrittenDate, bool) {
	year := 0
	if match[6] >= 0 {
		year = atoi(value[match[6]:match[7]])
	}
	day, ok := newCalendarDay(
		year,
		int(monthAbbreviations[strings.ToLower(value[match[monthGroup]:match[monthGroup]+3])]),
		atoi(value[match[dayGroup]:match[dayGroup+1]]),
	)
	if !ok {
		return WrittenDate{}, false
	}

	return WrittenDate{Day: day, Start: match[0], End: match[1]}, true
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}

	return d
}
