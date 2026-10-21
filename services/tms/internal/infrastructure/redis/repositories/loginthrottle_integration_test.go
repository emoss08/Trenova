//go:build integration

package repositories

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newIntegrationThrottle(
	t *testing.T,
	policy sessionports.LoginThrottlePolicy,
) sessionports.LoginThrottleStore {
	t.Helper()
	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })
	return NewLoginThrottleStoreWithPolicy(client, policy, zap.NewNop())
}

func TestLoginThrottleConcurrentAttemptsNeverExceedTheThreshold(t *testing.T) {
	policy := sessionports.DefaultLoginThrottlePolicy()
	store := newIntegrationThrottle(t, policy)
	key := sessionports.LoginThrottleKey{Account: "dana@example.com", ClientIP: "198.51.100.4"}

	const attempts = 50
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range attempts {
		wg.Go(func() {
			decision, err := store.Begin(t.Context(), key)
			if err == nil && !decision.Blocked {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()

	assert.Equal(t, int32(policy.AccountThreshold), allowed.Load())

	decision, err := store.Begin(t.Context(), key)
	require.NoError(t, err)
	assert.True(t, decision.Blocked)
	assert.Equal(t, sessionports.LoginThrottleScopeAccount, decision.Scope)
	assert.Positive(t, decision.RetryAfter)
}

func TestLoginThrottleTripsOnTheAttemptThatReachesTheThreshold(t *testing.T) {
	policy := sessionports.DefaultLoginThrottlePolicy()
	store := newIntegrationThrottle(t, policy)
	key := sessionports.LoginThrottleKey{Account: "dana@example.com", ClientIP: "198.51.100.4"}

	for attempt := 1; attempt < policy.AccountThreshold; attempt++ {
		decision, err := store.Begin(t.Context(), key)
		require.NoError(t, err)
		assert.False(t, decision.Blocked)
		assert.False(t, decision.Tripped)
		assert.Equal(t, attempt, decision.AccountFailures)
	}

	decision, err := store.Begin(t.Context(), key)
	require.NoError(t, err)
	assert.False(t, decision.Blocked)
	assert.True(t, decision.Tripped)
	assert.Equal(t, sessionports.LoginThrottleScopeAccount, decision.TrippedScope)
	assert.Equal(t, policy.AccountBaseLock, decision.RetryAfter)
}

func TestLoginThrottleSuccessClearsTheAccountAndReturnsTheIPAttempt(t *testing.T) {
	policy := sessionports.DefaultLoginThrottlePolicy()
	store := newIntegrationThrottle(t, policy)
	key := sessionports.LoginThrottleKey{Account: "dana@example.com", ClientIP: "198.51.100.4"}

	for range policy.AccountThreshold {
		_, err := store.Begin(t.Context(), key)
		require.NoError(t, err)
	}
	require.NoError(t, store.Succeed(t.Context(), key))

	decision, err := store.Begin(t.Context(), key)
	require.NoError(t, err)
	assert.False(t, decision.Blocked)
	assert.Equal(t, 1, decision.AccountFailures)
	assert.Equal(t, policy.AccountThreshold, decision.IPFailures)

	other := sessionports.LoginThrottleKey{Account: "colleague@example.com", ClientIP: key.ClientIP}
	for range 30 {
		_, err = store.Begin(t.Context(), other)
		require.NoError(t, err)
		require.NoError(t, store.Succeed(t.Context(), other))
	}
	decision, err = store.Begin(t.Context(), other)
	require.NoError(t, err)
	assert.False(t, decision.Blocked)
}

func TestLoginThrottleBlocksAnAddressPastItsThreshold(t *testing.T) {
	policy := sessionports.DefaultLoginThrottlePolicy()
	policy.IPThreshold = 3
	policy.IPWindow = time.Minute
	store := newIntegrationThrottle(t, policy)

	for i := range 3 {
		decision, err := store.Begin(t.Context(), sessionports.LoginThrottleKey{
			Account:  string(rune('a'+i)) + "@example.com",
			ClientIP: "198.51.100.9",
		})
		require.NoError(t, err)
		assert.False(t, decision.Blocked)
	}

	decision, err := store.Begin(t.Context(), sessionports.LoginThrottleKey{
		Account:  "z@example.com",
		ClientIP: "198.51.100.9",
	})
	require.NoError(t, err)
	assert.True(t, decision.Blocked)
	assert.Equal(t, sessionports.LoginThrottleScopeIP, decision.Scope)
}

func TestLoginThrottleLocksFollowThePolicysBackoff(t *testing.T) {
	policy := sessionports.DefaultLoginThrottlePolicy()
	policy.AccountWindow = time.Hour
	policy.IPThreshold = 0
	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })
	store := NewLoginThrottleStoreWithPolicy(client, policy, zap.NewNop())
	key := sessionports.LoginThrottleKey{Account: "dana@example.com", ClientIP: "198.51.100.4"}
	lockKey := loginThrottlePrefix + ":account:" + accountDigest(key.Account) + ":lock"

	for failures := 1; failures <= policy.AccountThreshold+25; failures++ {
		require.NoError(t, client.Del(t.Context(), lockKey).Err())
		decision, err := store.Begin(t.Context(), key)
		require.NoError(t, err)
		assert.Equal(t, failures, decision.AccountFailures)
		want := policy.AccountLockFor(failures)
		if want == 0 {
			assert.False(t, decision.Tripped, failures)
			continue
		}
		assert.True(t, decision.Tripped, failures)
		assert.Equal(t, want, decision.RetryAfter, failures)
	}
}
