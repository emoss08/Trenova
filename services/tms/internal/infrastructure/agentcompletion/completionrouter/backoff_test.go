package completionrouter

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

	assert.Equal(t, 500*time.Millisecond, retryDelay(0))
	assert.Equal(t, time.Second, retryDelay(1))
	assert.Equal(t, 2*time.Second, retryDelay(2))
	assert.Equal(t, 4*time.Second, retryDelay(3))
	assert.Equal(t, 5*time.Second, retryDelay(4))
	assert.Equal(t, 5*time.Second, retryDelay(40), "no overflow, whatever the attempt")
}

// A person who stopped a reply is not kept waiting for a backoff to elapse.
func TestWaitBeforeRetry_ReturnsAtOnceWhenCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	started := time.Now()
	err := waitBeforeRetry(ctx, 4)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(started), 100*time.Millisecond)
}
