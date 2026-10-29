package completionengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A 429 retried at once just spends the next request on the same limit.
func TestRetryDelay_DoublesFromHalfASecondAndStopsAtFive(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 500*time.Millisecond, RetryDelay(0))
	assert.Equal(t, time.Second, RetryDelay(1))
	assert.Equal(t, 2*time.Second, RetryDelay(2))
	assert.Equal(t, 4*time.Second, RetryDelay(3))
	assert.Equal(t, 5*time.Second, RetryDelay(4))
	assert.Equal(t, 5*time.Second, RetryDelay(40), "no overflow, whatever the attempt")
}

// A person who stopped a reply is not kept waiting for a backoff to elapse.
func TestPauseFor_ReturnsAtOnceWhenCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	started := time.Now()
	err := pauseFor(ctx, 4*time.Second)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(started), 100*time.Millisecond)
}
