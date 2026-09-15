//go:build integration

package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStore_EnforcesBurstAcrossClients_Integration(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	storeA, err := NewRedisStore(client, "test")
	require.NoError(t, err)
	storeB, err := NewRedisStore(client, "test")
	require.NoError(t, err)

	requests := []repositories.RateLimitRequest{
		{Key: "{org_1}:apikey:ak_1", Policy: policy(60, 2), Cost: 1},
		{Key: "{org_1}:tenant", Policy: policy(600, 100), Cost: 1},
	}

	first, err := storeA.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, first[0].Allowed)
	assert.Equal(t, 1, first[0].Remaining)
	assert.Equal(t, 2, first[0].Limit)

	second, err := storeB.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, second[0].Allowed)
	assert.Equal(t, 0, second[0].Remaining)

	third, err := storeA.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.False(
		t,
		third[0].Allowed,
		"the second replica's request counted against the same bucket",
	)
	assert.True(t, third[1].Allowed)
	assert.Greater(t, third[0].RetryAfter, time.Duration(0))
	assert.LessOrEqual(t, third[0].RetryAfter, time.Second)

	ttl, err := client.PTTL(t.Context(), "test:{org_1}:apikey:ak_1").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 2*time.Second)
}

func TestRedisStore_DeniedBatchDoesNotConsumeOtherScopes_Integration(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	store, err := NewRedisStore(client, "test")
	require.NoError(t, err)

	requests := []repositories.RateLimitRequest{
		{Key: "{org_2}:user:usr_1", Policy: policy(60, 10), Cost: 1},
		{Key: "{org_2}:tenant", Policy: policy(60, 1), Cost: 1},
	}

	_, err = store.Check(t.Context(), requests)
	require.NoError(t, err)

	denied, err := store.Check(t.Context(), requests)
	require.NoError(t, err)
	assert.True(t, denied[0].Allowed)
	assert.False(t, denied[1].Allowed)

	userOnly, err := store.Check(t.Context(), requests[:1])
	require.NoError(t, err)
	assert.Equal(t, 8, userOnly[0].Remaining)
}

func TestRedisStore_ConcurrentCallersNeverExceedBurst_Integration(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	store, err := NewRedisStore(client, "test")
	require.NoError(t, err)

	const (
		burst   = 5
		callers = 50
	)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			decisions, checkErr := store.Check(t.Context(), []repositories.RateLimitRequest{
				{Key: "{org_3}:apikey:ak_3", Policy: policy(60, burst), Cost: 1},
			})
			if checkErr != nil || !decisions[0].Allowed {
				return
			}
			mu.Lock()
			allowed++
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, burst, allowed)
}
