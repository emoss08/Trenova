package bcsession

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchLocksSerializeAndHonourCancellation(t *testing.T) {
	t.Parallel()

	locks := NewBatchLocks()
	release, err := locks.acquire(t.Context(), "batch")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err = locks.acquire(ctx, "batch")
	require.ErrorIs(t, err, context.DeadlineExceeded)

	other, err := locks.acquire(t.Context(), "other")
	require.NoError(t, err)
	other()

	release()
	again, err := locks.acquire(t.Context(), "batch")
	require.NoError(t, err)
	again()
	assert.Empty(t, locks.byBatch)
}
