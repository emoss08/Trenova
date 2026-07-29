package timeutils

import "fmt"

// FormatDurationMs renders a millisecond duration as a compact "3h 05m" label for
// dispatcher-facing explanations. Negative durations render as zero.
func FormatDurationMs(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	hours := ms / 3_600_000
	minutes := (ms % 3_600_000) / 60_000
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

// FormatLongDurationMs renders a millisecond duration the way a person says it,
// rolling hours into days once they stop being scannable: "45m", "3h 05m", "11d 12h".
// Negative durations render as zero.
func FormatLongDurationMs(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	days := ms / 86_400_000
	if days > 0 {
		hours := (ms % 86_400_000) / 3_600_000
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if ms >= 3_600_000 {
		return FormatDurationMs(ms)
	}
	return fmt.Sprintf("%dm", ms/60_000)
}
