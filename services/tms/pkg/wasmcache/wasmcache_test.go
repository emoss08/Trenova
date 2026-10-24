package wasmcache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheCacheIsKeyedByTheModuleAndPrivate(t *testing.T) {
	base := t.TempDir()

	cache, dir, err := Open(base, "sum")
	require.NoError(t, err)
	require.NoError(t, cache.Close(t.Context()))
	assert.Equal(t, filepath.Join(base, "sum"), dir)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(dirPerms), info.Mode().Perm())

	shared := filepath.Join(t.TempDir(), "shared")
	require.NoError(t, os.Mkdir(shared, 0o700))
	require.NoError(t, os.Chmod(shared, 0o777))
	cache, _, err = Open(shared, "sum")
	require.ErrorIs(t, err, ErrUnsafeDir)
	require.NotNil(t, cache)
	require.NoError(t, cache.Close(t.Context()))
}

func TestBaseDirReadsItsEnvironmentVariable(t *testing.T) {
	t.Setenv("TRENOVA_TEST_CACHE_DIR", "/somewhere")
	assert.Equal(t, "/somewhere", BaseDir("TRENOVA_TEST_CACHE_DIR", "name"))

	t.Setenv("TRENOVA_TEST_CACHE_DIR", "")
	assert.Equal(t, filepath.Join(os.TempDir(), "name"), BaseDir("TRENOVA_TEST_CACHE_DIR", "name"))
}
