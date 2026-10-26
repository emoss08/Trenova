//go:build integration

package redis_test

import (
	"testing"
	"time"

	"github.com/emoss08/keel/cache/rediscache"
	trenovaredis "github.com/emoss08/trenova/internal/infrastructure/redis"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ rediscache.Client = trenovaredis.BytesClient{}

func TestBytesClientStoresReadsAndDeletesValues(t *testing.T) {
	client := trenovaredis.NewBytesClient(sharedtestutil.SetupTestRedis(t))
	ctx := t.Context()

	_, found, err := client.Get(ctx, "bytes-client:missing")
	require.NoError(t, err)
	assert.False(t, found, "a missing key is a miss, not an error")

	require.NoError(t, client.Set(ctx, "bytes-client:k", []byte{0, 1, 2, 255}, time.Minute))
	value, found, err := client.Get(ctx, "bytes-client:k")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []byte{0, 1, 2, 255}, value)

	require.NoError(t, client.Del(ctx, "bytes-client:k"))
	_, found, err = client.Get(ctx, "bytes-client:k")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestBytesClientBacksTheKeelSharedCache(t *testing.T) {
	store := rediscache.New(
		trenovaredis.NewBytesClient(sharedtestutil.SetupTestRedis(t)),
		"trenova:keel:test:",
	)
	ctx := t.Context()

	require.NoError(t, store.Set(ctx, "entry", []byte("rows"), time.Minute))
	value, found, err := store.Get(ctx, "entry")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []byte("rows"), value)
}
