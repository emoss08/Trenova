//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupIdempotencyStore(t *testing.T) *idempotencyStore {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &idempotencyStore{client: client, l: zap.NewNop()}
}

func claimFor(key, owner string) repositories.IdempotencyClaim {
	return repositories.IdempotencyClaim{
		Key:         key,
		Fingerprint: "fp-1",
		Owner:       owner,
		LockTTL:     time.Minute,
	}
}

func TestIdempotencyStore_ClaimCompleteReplay_Integration(t *testing.T) {
	store := setupIdempotencyStore(t)

	first, err := store.Claim(t.Context(), claimFor("k1", "owner-a"))
	require.NoError(t, err)
	assert.True(t, first.Claimed)

	inFlight, err := store.Claim(t.Context(), claimFor("k1", "owner-b"))
	require.NoError(t, err)
	assert.False(t, inFlight.Claimed)
	require.NotNil(t, inFlight.Existing)
	assert.Equal(t, repositories.IdempotencyStateProcessing, inFlight.Existing.State)
	assert.Equal(t, "fp-1", inFlight.Existing.Fingerprint)

	body := []byte("{\"id\":\"shp_1\"}\x00binary")
	require.NoError(t, store.Complete(t.Context(), &repositories.IdempotencyCompletion{
		Key:         "k1",
		Owner:       "owner-a",
		Status:      201,
		ContentType: "application/json",
		Body:        body,
		TTL:         time.Hour,
	}))

	replay, err := store.Claim(t.Context(), claimFor("k1", "owner-c"))
	require.NoError(t, err)
	require.NotNil(t, replay.Existing)
	assert.Equal(t, repositories.IdempotencyStateCompleted, replay.Existing.State)
	assert.Equal(t, 201, replay.Existing.Status)
	assert.Equal(t, "application/json", replay.Existing.ContentType)
	assert.Equal(t, body, replay.Existing.Body)
	assert.False(t, replay.Existing.BodyOmitted)

	ttl, err := store.client.PTTL(t.Context(), idempotencyPrefix+"k1").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 59*time.Minute)
}

func TestIdempotencyStore_OnlyTheOwnerCompletesOrReleases_Integration(t *testing.T) {
	store := setupIdempotencyStore(t)

	_, err := store.Claim(t.Context(), claimFor("k2", "owner-a"))
	require.NoError(t, err)

	err = store.Complete(t.Context(), &repositories.IdempotencyCompletion{
		Key: "k2", Owner: "owner-b", Status: 200, TTL: time.Hour,
	})
	require.ErrorIs(t, err, repositories.ErrIdempotencyClaimLost)

	require.NoError(t, store.Release(t.Context(), "k2", "owner-b"))
	held, err := store.Claim(t.Context(), claimFor("k2", "owner-c"))
	require.NoError(t, err)
	assert.False(t, held.Claimed)

	require.NoError(t, store.Release(t.Context(), "k2", "owner-a"))
	reclaimed, err := store.Claim(t.Context(), claimFor("k2", "owner-c"))
	require.NoError(t, err)
	assert.True(t, reclaimed.Claimed)
}

func TestIdempotencyStore_CompletedRecordCannotBeReleased_Integration(t *testing.T) {
	store := setupIdempotencyStore(t)

	_, err := store.Claim(t.Context(), claimFor("k3", "owner-a"))
	require.NoError(t, err)
	require.NoError(t, store.Complete(t.Context(), &repositories.IdempotencyCompletion{
		Key: "k3", Owner: "owner-a", Status: 204, BodyOmitted: true, TTL: time.Hour,
	}))
	require.NoError(t, store.Release(t.Context(), "k3", "owner-a"))

	replay, err := store.Claim(t.Context(), claimFor("k3", "owner-b"))
	require.NoError(t, err)
	require.NotNil(t, replay.Existing)
	assert.Equal(t, 204, replay.Existing.Status)
	assert.True(t, replay.Existing.BodyOmitted)
	assert.Nil(t, replay.Existing.Body)
}

func TestIdempotencyStore_LockExpires_Integration(t *testing.T) {
	store := setupIdempotencyStore(t)

	claim := claimFor("k4", "owner-a")
	claim.LockTTL = 50 * time.Millisecond
	_, err := store.Claim(t.Context(), claim)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		result, claimErr := store.Claim(t.Context(), claimFor("k4", "owner-b"))
		return claimErr == nil && result.Claimed
	}, 2*time.Second, 25*time.Millisecond)
}
