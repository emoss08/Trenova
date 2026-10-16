package lrucache_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/lrucache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsNonPositiveSize(t *testing.T) {
	t.Parallel()

	cache, err := lrucache.New[string, int](0)
	require.Error(t, err)
	assert.Nil(t, cache)
}

func TestNewEvictsLeastRecentlyUsed(t *testing.T) {
	t.Parallel()

	cache, err := lrucache.New[string, int](2)
	require.NoError(t, err)

	assert.False(t, cache.Add("a", 1))
	assert.False(t, cache.Add("b", 2))
	_, ok := cache.Get("a")
	require.True(t, ok)
	assert.True(t, cache.Add("c", 3))

	_, ok = cache.Get("b")
	assert.False(t, ok, "b was the least recently used entry")
	value, ok := cache.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, value)
	assert.Equal(t, 2, cache.Len())
}

func TestCacheRemoveAndPurge(t *testing.T) {
	t.Parallel()

	cache, err := lrucache.New[string, int](4)
	require.NoError(t, err)

	cache.Add("a", 1)
	cache.Add("b", 2)

	assert.True(t, cache.Remove("a"))
	assert.False(t, cache.Remove("a"))
	assert.Equal(t, 1, cache.Len())

	cache.Purge()
	assert.Equal(t, 0, cache.Len())
}

func TestNewExpirableDropsExpiredEntries(t *testing.T) {
	t.Parallel()

	cache := lrucache.NewExpirable[string, int](4, 20*time.Millisecond)
	cache.Add("a", 1)

	value, ok := cache.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, value)

	require.Eventually(t, func() bool {
		_, ok := cache.Get("a")
		return !ok
	}, time.Second, 5*time.Millisecond)
}
