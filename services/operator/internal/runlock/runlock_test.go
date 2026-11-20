package runlock_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/runlock"
)

func TestASecondAgentOnTheSameStateIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	first, err := runlock.Acquire(dir)
	require.NoError(t, err)

	_, err = runlock.Acquire(dir)
	require.ErrorIs(t, err, runlock.ErrHeld)

	require.NoError(t, first.Release())
	again, err := runlock.Acquire(dir)
	require.NoError(t, err)
	require.NoError(t, again.Release())
}
