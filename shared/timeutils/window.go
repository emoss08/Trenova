package timeutils

import "time"

const windowTimeLayout = "Mon 02 Jan 2006 15:04 MST"

// WindowLabelUTC renders a scheduled window as human-readable UTC text: the
// end collapses to time-only when it falls on the start's day.
func WindowLabelUTC(start int64, end *int64) string {
	if start <= 0 {
		return ""
	}

	startLabel := time.Unix(start, 0).UTC().Format(windowTimeLayout)
	if end == nil || *end <= 0 {
		return startLabel
	}

	endTime := time.Unix(*end, 0).UTC()
	if endTime.Truncate(24*time.Hour) == time.Unix(start, 0).UTC().Truncate(24*time.Hour) {
		return startLabel + "–" + endTime.Format("15:04 MST")
	}
	return startLabel + " – " + endTime.Format(windowTimeLayout)
}
