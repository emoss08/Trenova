package timeutils

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	LocalDateTimeLayout        = "2006-01-02T15:04"
	LocalDateTimeSecondsLayout = "2006-01-02T15:04:05"
)

var (
	ErrNotLocalDateTime = errors.New("not a local date and time")
	ErrSkippedLocalTime = errors.New("local time does not exist")
)

var localDateTimeLayouts = []string{
	LocalDateTimeLayout,
	LocalDateTimeSecondsLayout,
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
}

func ParseLocalDateTime(value string, loc *time.Location) (int64, error) {
	if loc == nil {
		loc = time.UTC
	}

	text := strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return parsed.Unix(), nil
	}

	for _, layout := range localDateTimeLayouts {
		parsed, err := time.ParseInLocation(layout, text, loc)
		if err != nil {
			continue
		}
		if parsed.Format(layout) != text {
			return 0, fmt.Errorf(
				"%w: %s is skipped by the clock change in %s",
				ErrSkippedLocalTime, text, loc.String(),
			)
		}

		return parsed.Unix(), nil
	}

	return 0, fmt.Errorf(
		"%w: %q, expected a date and time such as 2026-10-01T08:00",
		ErrNotLocalDateTime, value,
	)
}

func FormatLocalDateTime(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}

	at := time.Unix(ts, 0).In(loc)
	if at.Second() != 0 {
		return at.Format(LocalDateTimeSecondsLayout)
	}

	return at.Format(LocalDateTimeLayout)
}

func ResolveZone(candidates ...string) (loc *time.Location, name string) {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" || trimmed == "Local" {
			continue
		}
		resolved, err := time.LoadLocation(trimmed)
		if err != nil {
			continue
		}

		return resolved, trimmed
	}

	return time.UTC, time.UTC.String()
}
