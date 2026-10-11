package common

import (
	"time"
)

const BackfillDayLayout = "2006-01-02"

func DayLabel(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format(BackfillDayLayout)
}
