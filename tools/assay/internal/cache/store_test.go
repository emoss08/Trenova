package cache

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sourcegraph/conc"
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

	var writers conc.WaitGroup
	for i := range 16 {
		writers.Go(func() {
			assert.NoError(t, store.Put(key(1), []byte("payload-"+strconv.Itoa(i))))
		})
	}
	writers.Wait()

	got, ok := store.Get(key(1))
	require.True(t, ok)
	assert.Contains(t, string(got), "payload-", "every write must land atomically")
}

func TestStorePrunesOldestEntries(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	for i := range GraphCapacity + 8 {
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
	assert.LessOrEqual(t, kept, GraphCapacity)
}

func TestNamespaceWithZeroCapacityNeverPrunes(t *testing.T) {
	root, err := NewStore(t.TempDir())
	require.NoError(t, err)

	store, err := root.Namespace("index", 0)
	require.NoError(t, err)

	const want = GraphCapacity + 40
	for i := range want {
		var fp Fingerprint
		fp[0] = byte(i)
		fp[1] = byte(i >> 8)
		require.NoError(t, store.Put(fp, []byte("payload")))
	}

	entries, err := os.ReadDir(store.Dir())
	require.NoError(t, err)

	var kept int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == entrySuffix {
			kept++
		}
	}
	assert.Equal(t, want, kept,
		"the coverage index holds one entry per package, so a capped store would evict records as fast as they are written")
}

func TestNamespaceIsIsolatedFromItsParent(t *testing.T) {
	root, err := NewStore(t.TempDir())
	require.NoError(t, err)
	child, err := root.Namespace("index", 0)
	require.NoError(t, err)

	require.NoError(t, root.Put(key(1), []byte("graph")))
	require.NoError(t, child.Put(key(1), []byte("record")))

	fromRoot, ok := root.Get(key(1))
	require.True(t, ok)
	fromChild, ok := child.Get(key(1))
	require.True(t, ok)

	assert.Equal(t, []byte("graph"), fromRoot)
	assert.Equal(t, []byte("record"), fromChild,
		"the same fingerprint must address different payloads in different namespaces")
}

func TestNamespaceOnNilStoreFails(t *testing.T) {
	var store *Store

	_, err := store.Namespace("index", 0)

	require.Error(t, err)
}

func TestStoreIgnoresUnreadableEntry(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	require.NoError(t, os.Mkdir(filepath.Join(dir, key(3).String()+entrySuffix), 0o755))

	_, ok := store.Get(key(3))

	assert.False(t, ok, "a corrupt entry must read as a miss, not an error")
}
