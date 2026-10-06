//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProviderBreaker(t *testing.T) *providerBreakerRepository {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &providerBreakerRepository{client: client}
}

func TestProviderBreaker_RestIsReadBackWithItsRemainingCooldown_Integration(t *testing.T) {
	repo := setupProviderBreaker(t)
	down := pulid.MustNew("aiprv_")
	up := pulid.MustNew("aiprv_")

	require.NoError(t, repo.Rest(t.Context(), down, time.Minute))

	resting, err := repo.Resting(t.Context(), []pulid.ID{down, up})
	require.NoError(t, err)
	require.Contains(t, resting, down)
	assert.NotContains(t, resting, up)
	assert.Greater(t, resting[down], 50*time.Second)
	assert.LessOrEqual(t, resting[down], time.Minute)

	ttl, err := repo.client.PTTL(t.Context(), "ai:breaker:"+down.String()).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
}

func TestProviderBreaker_ARestEndsWithItsCooldown_Integration(t *testing.T) {
	repo := setupProviderBreaker(t)
	id := pulid.MustNew("aiprv_")

	require.NoError(t, repo.Rest(t.Context(), id, 50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	resting, err := repo.Resting(t.Context(), []pulid.ID{id})
	require.NoError(t, err)
	assert.Empty(t, resting)
}

func TestProviderBreaker_NothingToReadIsNoTrip_Integration(t *testing.T) {
	repo := setupProviderBreaker(t)

	resting, err := repo.Resting(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, resting)
	require.NoError(t, repo.Rest(t.Context(), pulid.Nil, time.Minute))
	require.NoError(t, repo.Rest(t.Context(), pulid.MustNew("aiprv_"), 0))
}
