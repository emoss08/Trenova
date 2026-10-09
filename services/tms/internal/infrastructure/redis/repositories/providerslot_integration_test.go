//go:build integration

package repositories

import (
	"testing"
	"time"

	ports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProviderSlots(t *testing.T) *providerSlotRepository {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &providerSlotRepository{client: client}
}

func TestProviderSlots_HoldNoMoreThanTheLimit_Integration(t *testing.T) {
	repo := setupProviderSlots(t)
	provider := pulid.MustNew("aiprv_")

	for _, token := range []string{"a", "b"} {
		ok, err := repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
			ProviderID: provider, Token: token, Limit: 2, TTL: time.Minute,
		})
		require.NoError(t, err)
		assert.True(t, ok, token)
	}

	ok, err := repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "c", Limit: 2, TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.False(t, ok, "a third call waits for one of the two to finish")

	require.NoError(t, repo.Release(t.Context(), ports.ProviderSlotRequest{ProviderID: provider, Token: "a"}))

	ok, err = repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "c", Limit: 2, TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestProviderSlots_ExpireWhenTheirHolderStopsRefreshing_Integration(t *testing.T) {
	repo := setupProviderSlots(t)
	provider := pulid.MustNew("aiprv_")

	ok, err := repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "crashed", Limit: 1, TTL: 200 * time.Millisecond,
	})
	require.NoError(t, err)
	require.True(t, ok)

	time.Sleep(300 * time.Millisecond)

	held, err := repo.Refresh(t.Context(), ports.ProviderSlotRequest{
		ProviderID: provider, Token: "crashed", TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.False(t, held)

	ok, err = repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "next", Limit: 1, TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.True(t, ok, "a lease whose holder died frees its slot")
}

func TestProviderSlots_RefreshExtendsAHeldLease_Integration(t *testing.T) {
	repo := setupProviderSlots(t)
	provider := pulid.MustNew("aiprv_")

	ok, err := repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "live", Limit: 1, TTL: 300 * time.Millisecond,
	})
	require.NoError(t, err)
	require.True(t, ok)

	held, err := repo.Refresh(t.Context(), ports.ProviderSlotRequest{
		ProviderID: provider, Token: "live", TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.True(t, held)

	time.Sleep(400 * time.Millisecond)

	ok, err = repo.Acquire(t.Context(), ports.AcquireProviderSlotRequest{
		ProviderID: provider, Token: "other", Limit: 1, TTL: time.Minute,
	})
	require.NoError(t, err)
	assert.False(t, ok, "a refreshed lease still holds its slot")
}
