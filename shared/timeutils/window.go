package timeutils

import "time"

const windowTimeLayout = "Mon 02 Jan 2006 15:04 MST"

// WindowLabelUTC renders a scheduled window as human-readable UTC text: the
// end collapses to time-only when it falls on the start's day.
func WindowLabelUTC(start int64, end *int64) string {
	return WindowLabelIn(start, end, time.UTC)
}

// WindowLabelIn renders a scheduled window in the given zone, collapsing the
// end to time-only when it falls on the start's day there. A nil zone is UTC.
func WindowLabelIn(start int64, end *int64, loc *time.Location) string {
	if start <= 0 {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}

	startTime := time.Unix(start, 0).In(loc)
	startLabel := startTime.Format(windowTimeLayout)
	if end == nil || *end <= 0 {
		return startLabel
	}

	endTime := time.Unix(*end, 0).In(loc)
	if endTime.Year() == startTime.Year() && endTime.YearDay() == startTime.YearDay() {
		return startLabel + "–" + endTime.Format("15:04 MST")
	}
	return startLabel + " – " + endTime.Format(windowTimeLayout)
}
