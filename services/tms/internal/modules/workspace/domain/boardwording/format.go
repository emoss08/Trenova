package boardwording

import "strconv"

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
)

const hoursUpTo = 3 * secondsPerDay

func FormatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = -seconds
	}
	if seconds >= hoursUpTo {
		days := seconds / secondsPerDay
		hours := seconds % secondsPerDay / secondsPerHour
		if hours > 0 {
			return strconv.FormatInt(days, 10) + "d " + strconv.FormatInt(hours, 10) + "h"
		}
		return strconv.FormatInt(days, 10) + "d"
	}

	return FormatHours(seconds)
}

func FormatHours(seconds int64) string {
	if seconds < 0 {
		seconds = -seconds
	}
	hours := seconds / secondsPerHour
	minutes := seconds % secondsPerHour / secondsPerMinute
	switch {
	case hours > 0 && minutes > 0:
		return strconv.FormatInt(hours, 10) + "h " + strconv.FormatInt(minutes, 10) + "m"
	case hours > 0:
		return strconv.FormatInt(hours, 10) + "h"
	default:
		return strconv.FormatInt(minutes, 10) + "m"
	}
}
