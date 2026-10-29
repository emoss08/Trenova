package completionengine

import (
	"time"
)

func BackgroundLatency(submittedAt int64, now time.Time) time.Duration {
	if submittedAt <= 0 {
		return 0
	}

	return max(now.Sub(time.Unix(submittedAt, 0)), 0)
}
