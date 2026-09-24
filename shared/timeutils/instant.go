package timeutils

import (
	"strings"
	"time"
)

const millisecondThreshold = 1e12

var instantLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"01/02/2006",
}

func ParseInstant(v any) (int64, bool) {
	switch value := v.(type) {
	case int64:
		return fromEpoch(value), true
	case int:
		return fromEpoch(int64(value)), true
	case float64:
		return fromEpoch(int64(value)), true
	case string:
		return parseInstantText(strings.TrimSpace(value))
	default:
		return 0, false
	}
}

func fromEpoch(value int64) int64 {
	if value > millisecondThreshold || value < -millisecondThreshold {
		return value / 1000
	}

	return value
}

func parseInstantText(value string) (int64, bool) {
	if value == "" {
		return 0, false
	}
	if parsed, ok := ParseTimeRFC3339(value); ok {
		return parsed.Unix(), true
	}
	for _, layout := range instantLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Unix(), true
		}
	}

	return 0, false
}
