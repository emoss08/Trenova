package timeutils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Window struct {
	Start int64
	End   int64
}

var (
	clockPattern    = regexp.MustCompile(`(?i)\b(\d{1,2})(?::(\d{2}))?\s*(am|pm|a\.m\.|p\.m\.)?\b`)
	isoClockPattern = regexp.MustCompile(`T(\d{2}):(\d{2})`)
	datePattern     = regexp.MustCompile(
		`(?i)\b\d{4}-\d{1,2}-\d{1,2}\b|\b\d{1,2}/\d{1,2}(?:/(?:\d{4}|\d{2}))?\b|` +
			`\b(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?\s+\d{1,2}(?:st|nd|rd|th)?(?:,?\s+\d{4})?\b`,
	)
)

const (
	hoursPerHalfDay = 12
	maxHour         = 23
	maxMinute       = 59
	staleWindowDays = 30
	maxClockMatches = 2
)

func ParseWindow(text string, loc *time.Location, now time.Time) (Window, bool) {
	if loc == nil {
		loc = time.UTC
	}

	day, ok := FindDocumentDate(text)
	if !ok {
		return Window{}, false
	}

	year := day.Year
	if !day.HasYear() {
		year = inferYear(day, now.In(loc))
	}

	rest := datePattern.ReplaceAllString(text, " ")
	clocks := findClocks(text, rest)
	startHour, startMinute := 0, 0
	if len(clocks) > 0 {
		startHour, startMinute = clocks[0][0], clocks[0][1]
	}

	start := time.Date(year, day.Month, day.Day, startHour, startMinute, 0, 0, loc)
	window := Window{Start: start.Unix()}
	if len(clocks) > 1 {
		end := time.Date(year, day.Month, day.Day, clocks[1][0], clocks[1][1], 0, 0, loc)
		if end.Before(start) {
			end = end.AddDate(0, 0, 1)
		}
		window.End = end.Unix()
	}

	return window, true
}

func inferYear(day CalendarDay, now time.Time) int {
	candidate := time.Date(now.Year(), day.Month, day.Day, 0, 0, 0, 0, now.Location())
	if candidate.Before(now.AddDate(0, 0, -staleWindowDays)) {
		return now.Year() + 1
	}

	return now.Year()
}

func findClocks(original, withoutDates string) [][2]int {
	if match := isoClockPattern.FindStringSubmatch(original); match != nil {
		hour, _ := strconv.Atoi(match[1])
		minute, _ := strconv.Atoi(match[2])
		if hour <= maxHour && minute <= maxMinute {
			return [][2]int{{hour, minute}}
		}
	}

	clocks := make([][2]int, 0, maxClockMatches)
	for _, match := range clockPattern.FindAllStringSubmatch(withoutDates, -1) {
		if len(clocks) == maxClockMatches {
			break
		}
		if match[2] == "" && match[3] == "" {
			continue
		}

		hour, _ := strconv.Atoi(match[1])
		minute := 0
		if match[2] != "" {
			minute, _ = strconv.Atoi(match[2])
		}
		switch strings.ToLower(strings.ReplaceAll(match[3], ".", "")) {
		case "pm":
			if hour < hoursPerHalfDay {
				hour += hoursPerHalfDay
			}
		case "am":
			if hour == hoursPerHalfDay {
				hour = 0
			}
		}
		if hour > maxHour || minute > maxMinute {
			continue
		}
		clocks = append(clocks, [2]int{hour, minute})
	}

	return clocks
}
