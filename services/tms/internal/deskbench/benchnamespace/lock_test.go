//go:build unix

package benchnamespace

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
	first, err := Acquire(dir, "deskbench")
	require.NoError(t, err)

	holder, err := os.ReadFile(filepath.Join(dir, "deskbench.lock"))
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(holder))

	_, err = Acquire(dir, "deskbench")
	require.ErrorIs(t, err, ErrNamespaceBusy)
	assert.Contains(t, err.Error(), "--namespace")

	other, err := Acquire(dir, "deskbench-two")
	require.NoError(t, err, "another namespace is its own lock")
	require.NoError(t, other.Release())

	require.NoError(t, first.Release())
	again, err := Acquire(dir, "deskbench")
	require.NoError(t, err)
	require.NoError(t, again.Release())
}
