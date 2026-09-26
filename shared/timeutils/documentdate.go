package timeutils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CalendarDay struct {
	Year  int
	Month time.Month
	Day   int
}

var (
	isoDayPattern     = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})(?:\D|$)`)
	numericDayPattern = regexp.MustCompile(`\b(\d{1,2})[/-](\d{1,2})(?:[/-](\d{2}|\d{4}))?\b`)
	namedDayPattern   = regexp.MustCompile(
		`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?\b`,
	)
	monthAbbreviations = map[string]time.Month{
		"jan": time.January, "feb": time.February, "mar": time.March, "apr": time.April,
		"may": time.May, "jun": time.June, "jul": time.July, "aug": time.August,
		"sep": time.September, "sept": time.September, "oct": time.October,
		"nov": time.November, "dec": time.December,
	}
)

const twoDigitYearBase = 2000

func FindDocumentDate(value string) (CalendarDay, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return CalendarDay{}, false
	}

	if match := isoDayPattern.FindStringSubmatch(text); match != nil {
		return newCalendarDay(atoi(match[1]), atoi(match[2]), atoi(match[3]))
	}

	if match := namedDayPattern.FindStringSubmatch(text); match != nil {
		month := monthAbbreviations[strings.ToLower(match[1])]
		return newCalendarDay(atoi(match[3]), int(month), atoi(match[2]))
	}

	if match := numericDayPattern.FindStringSubmatch(text); match != nil {
		year := atoi(match[3])
		if len(match[3]) == 2 {
			year += twoDigitYearBase
		}
		return newCalendarDay(year, atoi(match[1]), atoi(match[2]))
	}

	return CalendarDay{}, false
}

func (d CalendarDay) HasYear() bool {
	return d.Year > 0
}

func (d CalendarDay) Matches(t time.Time) bool {
	if t.Month() != d.Month || t.Day() != d.Day {
		return false
	}

	return !d.HasYear() || t.Year() == d.Year
}

func (d CalendarDay) String() string {
	if !d.HasYear() {
		return time.Date(0, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Format("01-02")
	}

	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Format(ISODateLayout)
}

func newCalendarDay(year, month, day int) (CalendarDay, bool) {
	if month < 1 || month > 12 || day < 1 {
		return CalendarDay{}, false
	}

	checkYear := year
	if checkYear <= 0 {
		checkYear = 2000
	}
	if day > time.Date(checkYear, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day() {
		return CalendarDay{}, false
	}

	return CalendarDay{Year: year, Month: time.Month(month), Day: day}, true
}

func atoi(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return parsed
}
