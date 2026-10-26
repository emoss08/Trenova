package reportengine

import (
	"testing"
	"time"

	"github.com/emoss08/keel/analytics/cache"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTieredStoreWritesBothTiersAndPromotesASharedHit(t *testing.T) {
	t.Parallel()

	local := cache.NewMemory(cache.MemoryOptions{MaxBytes: 1 << 20})
	shared := cache.NewMemory(cache.MemoryOptions{MaxBytes: 1 << 20})
	store := &tieredStore{local: local, shared: shared, ttl: time.Minute}

	require.NoError(t, store.Set(t.Context(), "a", []byte("rows"), time.Minute))
	_, inLocal, err := local.Get(t.Context(), "a")
	require.NoError(t, err)
	_, inShared, err := shared.Get(t.Context(), "a")
	require.NoError(t, err)
	assert.True(t, inLocal)
	assert.True(t, inShared)

	require.NoError(t, shared.Set(t.Context(), "b", []byte("other"), time.Minute))
	value, ok, err := store.Get(t.Context(), "b")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("other"), value)
	_, promoted, err := local.Get(t.Context(), "b")
	require.NoError(t, err)
	assert.True(t, promoted, "a hit in the shared tier is kept in this process too")

	_, ok, err = store.Get(t.Context(), "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestTieredStoreKeepsAValueTooLargeForThisProcessInTheSharedTier(t *testing.T) {
	t.Parallel()

	local := cache.NewMemory(cache.MemoryOptions{MaxBytes: 128})
	shared := cache.NewMemory(cache.MemoryOptions{MaxBytes: 1 << 20})
	store := &tieredStore{local: local, shared: shared, ttl: time.Minute}

	require.NoError(t, store.Set(t.Context(), "big", make([]byte, 4096), time.Minute))
	_, inShared, err := shared.Get(t.Context(), "big")
	require.NoError(t, err)
	assert.True(t, inShared)
}

func TestResultStoreKeepsReportRowsInProcessUnlessTheSharedTierIsOn(t *testing.T) {
	t.Parallel()

	off := &Engine{cfg: &config.ReportingConfig{}}
	assert.Nil(t, off.resultStore(), "a disabled result cache gives the runs no store")

	local := &Engine{cfg: &config.ReportingConfig{
		ResultCache: config.ReportResultCacheConfig{Enabled: true},
	}}
	_, inProcess := local.resultStore().(*cache.Memory)
	assert.True(t, inProcess, "the default keeps report rows in this process only")

	sharedWithoutRedis := &Engine{cfg: &config.ReportingConfig{
		ResultCache: config.ReportResultCacheConfig{
			Enabled: true,
			Shared:  config.ReportSharedResultCacheConfig{Enabled: true},
		},
	}}
	_, inProcess = sharedWithoutRedis.resultStore().(*cache.Memory)
	assert.True(t, inProcess, "without Redis the shared tier has nowhere to go")
}
