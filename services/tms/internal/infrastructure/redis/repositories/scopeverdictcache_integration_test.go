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

func setupScopeVerdictCache(t *testing.T) *scopeVerdictCacheRepository {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &scopeVerdictCacheRepository{client: client, l: zap.NewNop()}
}

func TestScopeVerdictCache_SetAndGet_Integration(t *testing.T) {
	repo := setupScopeVerdictCache(t)

	verdict := &repositories.CachedScopeVerdict{Category: "TransportationOperations", Reasoning: "asks about drivers"}
	require.NoError(t, repo.Set(t.Context(), "org_1:abc", verdict, time.Minute))

	got, err := repo.Get(t.Context(), "org_1:abc")
	require.NoError(t, err)
	assert.Equal(t, verdict, got)
}

func TestScopeVerdictCache_MissIsNilAndNoError_Integration(t *testing.T) {
	repo := setupScopeVerdictCache(t)

	got, err := repo.Get(t.Context(), "org_1:missing")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestScopeVerdictCache_Expires_Integration(t *testing.T) {
	repo := setupScopeVerdictCache(t)

	require.NoError(t, repo.Set(t.Context(), "org_1:short", &repositories.CachedScopeVerdict{Category: "x"}, time.Second))
	time.Sleep(1500 * time.Millisecond)

	got, err := repo.Get(t.Context(), "org_1:short")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// A row this replica cannot decode is dropped rather than left to trip every
// reader, so the next classification replaces it.
func TestScopeVerdictCache_DropsAnUndecodableRow_Integration(t *testing.T) {
	repo := setupScopeVerdictCache(t)
	require.NoError(t, repo.client.Set(t.Context(), repo.key("org_1:bad"), "not json", time.Minute).Err())

	_, err := repo.Get(t.Context(), "org_1:bad")
	require.Error(t, err)

	exists, err := repo.client.Exists(t.Context(), repo.key("org_1:bad")).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}
