//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupQueryVectorCache(t *testing.T) *queryVectorCache {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &queryVectorCache{client: client, l: zap.NewNop()}
}

func queryVectorKey() agentports.QueryVectorCacheKey {
	return agentports.QueryVectorCacheKey{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ModelKey:   "nomic-embed-text",
		TextHash:   "abc123",
	}
}

func unitVector(dimensions int) []float32 {
	vector := make([]float32, dimensions)
	vector[0] = 1

	return vector
}

func TestQueryVectorCache_RoundTripsAVector_Integration(t *testing.T) {
	cache := setupQueryVectorCache(t)
	key := queryVectorKey()
	vector := agentports.QueryVector{
		Available:  true,
		Vector:     unitVector(airetrieval.Dimensions768),
		ModelKey:   key.ModelKey,
		Dimensions: airetrieval.Dimensions768,
	}

	require.NoError(t, cache.Set(t.Context(), key, vector, time.Minute))
	got, ok, err := cache.Get(t.Context(), key)

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, vector, got)
}

func TestQueryVectorCache_KeepsTenantsApart_Integration(t *testing.T) {
	cache := setupQueryVectorCache(t)
	key := queryVectorKey()
	require.NoError(t, cache.Set(t.Context(), key, agentports.QueryVector{
		Available:  true,
		Vector:     unitVector(airetrieval.Dimensions768),
		ModelKey:   key.ModelKey,
		Dimensions: airetrieval.Dimensions768,
	}, time.Minute))

	other := key
	other.TenantInfo.OrgID = pulid.MustNew("org_")
	_, ok, err := cache.Get(t.Context(), other)

	require.NoError(t, err)
	assert.False(t, ok)
}

func TestQueryVectorCache_DropsAnUnreadableEntry_Integration(t *testing.T) {
	cache := setupQueryVectorCache(t)
	key := queryVectorKey()
	require.NoError(t, cache.client.Set(t.Context(), cache.key(key), "not json", time.Minute).Err())

	_, ok, err := cache.Get(t.Context(), key)

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, cache.client.Exists(t.Context(), cache.key(key)).Val())
}

func TestQueryVectorCache_StoresNothingUnusable_Integration(t *testing.T) {
	cache := setupQueryVectorCache(t)
	key := queryVectorKey()

	require.NoError(t, cache.Set(t.Context(), key, agentports.QueryVector{}, time.Minute))

	assert.Zero(t, cache.client.Exists(t.Context(), cache.key(key)).Val())
}
