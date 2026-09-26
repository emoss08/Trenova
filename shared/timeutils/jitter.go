package timeutils

import (
	"math/rand/v2"
	"time"
)

// Jittered spreads a duration by ten percent either way, so many timers set
// from the same moment (every stream a fleet restart reopened) do not all fire
// in the same second.
func Jittered(d time.Duration) time.Duration {
	spread := int64(d / 10)
	if spread <= 0 {
		return d
	}
	offset := rand.Int64N(2 * spread) //nolint:gosec // scheduling jitter, not security

	return d - time.Duration(spread) + time.Duration(offset)
}
