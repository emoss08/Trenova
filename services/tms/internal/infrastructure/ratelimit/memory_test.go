package ratelimit

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_CommitsOnlyWhenEveryScopeAllows(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0)
	store := NewMemoryStore(MemoryStoreOptions{Now: func() time.Time { return current }})

	requests := []repositories.RateLimitRequest{
		{Key: "principal", Policy: policy(60, 10), Cost: 1},
		{Key: "tenant", Policy: policy(60, 1), Cost: 1},
	}

	first, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, first[0].Allowed)
	assert.True(t, first[1].Allowed)
	assert.Equal(t, 9, first[0].Remaining)

	second, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, second[0].Allowed)
	assert.False(t, second[1].Allowed)
	assert.Equal(t, time.Second, second[1].RetryAfter)

	third, err := store.Check(t.Context(), requests[:1])
	require.NoError(t, err)
	assert.Equal(t, 8, third[0].Remaining,
		"the principal token was not consumed when the tenant scope denied")
}

func TestMemoryStore_RefillsOverTime(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0)
	store := NewMemoryStore(MemoryStoreOptions{Now: func() time.Time { return current }})
	requests := []repositories.RateLimitRequest{{Key: "k", Policy: policy(60, 1), Cost: 1}}

	first, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, first[0].Allowed)

	denied, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.False(t, denied[0].Allowed)

	current = current.Add(time.Second)
	allowed, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, allowed[0].Allowed)
}

func TestMemoryStore_SweepsFullBuckets(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0)
	store := NewMemoryStore(MemoryStoreOptions{
		CleanupInterval: time.Second,
		Now:             func() time.Time { return current },
	})

	_, err := store.Check(t.Context(), []repositories.RateLimitRequest{
		{Key: "a", Policy: policy(60, 1), Cost: 1},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, store.Len())

	current = current.Add(2 * time.Second)
	_, err = store.Check(t.Context(), []repositories.RateLimitRequest{
		{Key: "b", Policy: policy(60, 1), Cost: 1},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, store.Len(), "bucket a refilled and was swept")
}

func TestMemoryStore_RejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore(MemoryStoreOptions{})
	_, err := store.Check(t.Context(), []repositories.RateLimitRequest{{Key: ""}})
	assert.ErrorIs(t, err, ErrEmptyKey)
}
