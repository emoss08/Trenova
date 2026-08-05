package cache

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func key(n byte) Fingerprint {
	var fp Fingerprint
	fp[0] = n

	return fp
}

func TestStoreRoundTrip(t *testing.T) {
	store, err := NewStore(t.TempDir())
	require.NoError(t, err)

	payload := []byte("graph bytes")
	require.NoError(t, store.Put(key(1), payload))

	got, ok := store.Get(key(1))
	require.True(t, ok)
	assert.Equal(t, payload, got)
}

func TestStoreMissReturnsFalse(t *testing.T) {
	store, err := NewStore(t.TempDir())
	require.NoError(t, err)

	_, ok := store.Get(key(9))

	assert.False(t, ok)
}

func TestStoreOverwritesExistingEntry(t *testing.T) {
	store, err := NewStore(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, store.Put(key(1), []byte("first")))
	require.NoError(t, store.Put(key(1), []byte("second")))

	got, ok := store.Get(key(1))
	require.True(t, ok)
	assert.Equal(t, []byte("second"), got)
}

func TestStoreCreatesMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "assay")

	store, err := NewStore(dir)
	require.NoError(t, err)

	assert.Equal(t, dir, store.Dir())
	assert.DirExists(t, dir)
}

func TestStoreLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	require.NoError(t, store.Put(key(1), []byte("payload")))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.Equal(t, entrySuffix, filepath.Ext(entry.Name()),
			"writes must commit by rename, leaving no partial files")
	}
}

func TestStoreConcurrentPutsAreSafe(t *testing.T) {
	store, err := NewStore(t.TempDir())
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			assert.NoError(t, store.Put(key(1), []byte("payload-"+strconv.Itoa(n))))
		}(i)
	}
	wg.Wait()

	got, ok := store.Get(key(1))
	require.True(t, ok)
	assert.Contains(t, string(got), "payload-", "every write must land atomically")
}

func TestStorePrunesOldestEntries(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	for i := range maxEntries + 8 {
		var fp Fingerprint
		fp[0] = byte(i)
		fp[1] = byte(i >> 8)
		require.NoError(t, store.Put(fp, []byte("payload")))
	}

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var kept int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == entrySuffix {
			kept++
		}
	}
	assert.LessOrEqual(t, kept, maxEntries)
}

func TestStoreIgnoresUnreadableEntry(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	require.NoError(t, os.Mkdir(filepath.Join(dir, key(3).String()+entrySuffix), 0o755))

	_, ok := store.Get(key(3))

	assert.False(t, ok, "a corrupt entry must read as a miss, not an error")
}
