package gqlctx

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionMemo_RoundTrip(t *testing.T) {
	t.Parallel()

	memo := NewPermissionMemo()

	_, found := memo.Lookup("shipment|read")
	assert.False(t, found)

	memo.Store("shipment|read", true)
	memo.Store("shipment|delete", false)

	allowed, found := memo.Lookup("shipment|read")
	assert.True(t, found)
	assert.True(t, allowed)

	allowed, found = memo.Lookup("shipment|delete")
	assert.True(t, found, "denials are memoised too; permissions do not change mid-request")
	assert.False(t, allowed)

	assert.Equal(t, 2, memo.Len())
}

func TestPermissionMemo_Context(t *testing.T) {
	t.Parallel()

	_, ok := PermissionMemoFrom(t.Context())
	assert.False(t, ok, "a bare context carries no memo")

	memo := NewPermissionMemo()
	ctx := WithPermissionMemo(t.Context(), memo)

	got, ok := PermissionMemoFrom(ctx)
	require.True(t, ok)
	assert.Same(t, memo, got)

	var nilMemo *PermissionMemo
	_, ok = PermissionMemoFrom(WithPermissionMemo(t.Context(), nilMemo))
	assert.False(t, ok, "a nil memo must read as absent rather than panic later")
}

func TestPermissionMemo_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	memo := NewPermissionMemo()

	var wg sync.WaitGroup
	for i := range 200 {
		wg.Add(2)
		key := "resource|" + strconv.Itoa(i%10)
		go func() {
			defer wg.Done()
			memo.Store(key, i%2 == 0)
		}()
		go func() {
			defer wg.Done()
			memo.Lookup(key)
		}()
	}
	wg.Wait()

	assert.Equal(t, 10, memo.Len())
}
