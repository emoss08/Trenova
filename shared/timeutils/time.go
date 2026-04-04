package timeutils

import (
	"fmt"
	"strconv"
	"time"
)

func NowUnix() int64 {
	return time.Now().Unix()
}

func WithDefaultDuration(val, def time.Duration) time.Duration {
	if val == 0 {
		return def
	}

	return val
}

func ParseTimeRFC3339(value string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, true
	}
	if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		if timestamp > 1e12 {
			return time.Unix(timestamp/1000, (timestamp%1000)*1e6), true
		}
		return time.Unix(timestamp, 0), true
	}
	return time.Time{}, false
}

func UnixToHumanReadable(ts int64) string {
	return time.Unix(ts, 0).Format("January 2, 2006 3:04:05 PM MST")
}

func NormalizeTimezone(timezone string) string {
	if timezone == "" {
		return "UTC"
	}

	return timezone
}

func NowAddDuration(duration time.Duration) int64 {
	return time.Now().Add(duration).Unix()
}

func DayStartUnix(ts int64, timezone string) (int64, error) {
	if ts <= 0 {
		return 0, fmt.Errorf("timestamp must be greater than zero")
	}

	loc, err := time.LoadLocation(NormalizeTimezone(timezone))
	if err != nil {
		return 0, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	t := time.Unix(ts, 0).In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start.Unix(), nil
}

func DayEndUnix(ts int64, timezone string) (int64, error) {
	if ts <= 0 {
		return 0, fmt.Errorf("timestamp must be greater than zero")
	}

	loc, err := time.LoadLocation(NormalizeTimezone(timezone))
	if err != nil {
		return 0, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	t := time.Unix(ts, 0).In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1).Add(-time.Second)
	return end.Unix(), nil
}

func CurrentDateInTimezone(location string) string {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func TimeZoneAwareNow(location string) int64 {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return NowUnix()
	}
	now := time.Now().In(loc)
	return now.Unix()
}
