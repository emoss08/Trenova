//go:build unix

package deskbench

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockNamespace_RefusesASecondBenchUntilTheFirstLetsGo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first, err := lockNamespace(dir, "deskbench")
	require.NoError(t, err)

	holder, err := os.ReadFile(filepath.Join(dir, "deskbench.lock"))
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(holder))

	_, err = lockNamespace(dir, "deskbench")
	require.ErrorIs(t, err, ErrNamespaceBusy)
	assert.Contains(t, err.Error(), "--namespace")

	other, err := lockNamespace(dir, "deskbench-two")
	require.NoError(t, err, "another namespace is its own lock")
	require.NoError(t, other.release())

	require.NoError(t, first.release())
	again, err := lockNamespace(dir, "deskbench")
	require.NoError(t, err)
	require.NoError(t, again.release())
}
