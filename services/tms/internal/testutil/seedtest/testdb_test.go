package seedtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollUntilAcquired_ReturnsOnceTheLockIsFree(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := pollUntilAcquired(t.Context(), time.Minute, func() (bool, error) {
		attempts++
		return attempts == 3, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts, "polling should stop at the first successful attempt")
}

func TestPollUntilAcquired_KeepsWaitingRatherThanFailingOnContention(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := pollUntilAcquired(t.Context(), 200*time.Millisecond, func() (bool, error) {
		attempts++
		return false, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Greater(t, attempts, 1, "contention must be retried, not reported on the first miss")
}

func TestPollUntilAcquired_SurfacesAttemptErrorsImmediately(t *testing.T) {
	t.Parallel()

	attemptErr := errors.New("connection reset")

	attempts := 0
	err := pollUntilAcquired(t.Context(), time.Minute, func() (bool, error) {
		attempts++
		return false, attemptErr
	})

	require.ErrorIs(t, err, attemptErr)
	assert.Equal(t, 1, attempts)
}

func TestPollUntilAcquired_StopsWhenTheContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	attempts := 0
	err := pollUntilAcquired(ctx, time.Minute, func() (bool, error) {
		attempts++
		cancel()
		return false, nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
}
