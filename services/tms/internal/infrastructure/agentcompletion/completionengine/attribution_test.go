package completionengine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackgroundLatency(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_100, 0)

	assert.Equal(t, 100*time.Second, BackgroundLatency(1_700_000_000, now))
	assert.Zero(t, BackgroundLatency(0, now))
	assert.Zero(t, BackgroundLatency(1_700_000_200, now),
		"a clock that ran backwards is not a negative wait")
}
