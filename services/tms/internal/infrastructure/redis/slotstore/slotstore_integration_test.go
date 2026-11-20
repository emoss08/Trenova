//go:build integration

package slotstore_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/redis/slotstore"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStore(t *testing.T) *slotstore.Store {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return slotstore.New(client)
}

func claim(token string, ttl time.Duration, bounds ...slotstore.Bound) slotstore.Claim {
	return slotstore.Claim{Bounds: bounds, Token: token, TTL: ttl}
}

func TestStore_TakesASlotInEveryBoundOrInNone_Integration(t *testing.T) {
	store := setupStore(t)
	instance := slotstore.Bound{Key: "test:instance", Limit: 2}
	orgA := slotstore.Bound{Key: "test:org:a", Limit: 1}
	orgB := slotstore.Bound{Key: "test:org:b", Limit: 1}

	ok, err := store.Acquire(t.Context(), claim("a1", time.Minute, instance, orgA))
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = store.Acquire(t.Context(), claim("a2", time.Minute, instance, orgA))
	require.NoError(t, err)
	assert.False(t, ok, "the organization's bound refuses a second slot")

	ok, err = store.Acquire(t.Context(), claim("b1", time.Minute, instance, orgB))
	require.NoError(t, err)
	assert.True(t, ok, "a refused claim took nothing from the instance bound")

	ok, err = store.Acquire(t.Context(), claim("b2", time.Minute, instance, orgB))
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, store.Release(t.Context(), claim("a1", time.Minute, instance, orgA)))

	ok, err = store.Acquire(t.Context(), claim("a2", time.Minute, instance, orgA))
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestStore_ASlotLapsesUnlessRefreshed_Integration(t *testing.T) {
	store := setupStore(t)
	bound := slotstore.Bound{Key: "test:lapse", Limit: 1}

	ok, err := store.Acquire(t.Context(), claim("held", 300*time.Millisecond, bound))
	require.NoError(t, err)
	require.True(t, ok)

	held, err := store.Refresh(t.Context(), claim("held", 300*time.Millisecond, bound))
	require.NoError(t, err)
	assert.True(t, held)

	ok, err = store.Acquire(t.Context(), claim("next", time.Minute, bound))
	require.NoError(t, err)
	assert.False(t, ok)

	time.Sleep(400 * time.Millisecond)

	ok, err = store.Acquire(t.Context(), claim("next", time.Minute, bound))
	require.NoError(t, err)
	assert.True(t, ok, "a holder that stopped refreshing frees its slot")

	held, err = store.Refresh(t.Context(), claim("held", time.Minute, bound))
	require.NoError(t, err)
	assert.False(t, held)
}

func TestStore_RefusesAnIncompleteClaim_Integration(t *testing.T) {
	store := setupStore(t)

	_, err := store.Acquire(
		t.Context(),
		claim("", time.Minute, slotstore.Bound{Key: "k", Limit: 1}),
	)
	require.ErrorIs(t, err, slotstore.ErrIncompleteClaim)

	_, err = store.Acquire(t.Context(), claim("t", time.Minute, slotstore.Bound{Key: "k"}))
	require.ErrorIs(t, err, slotstore.ErrIncompleteClaim)
}
