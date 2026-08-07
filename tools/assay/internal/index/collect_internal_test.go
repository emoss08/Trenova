package index

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunTestRefusesToFabricateCoverageWhenCancelled pins the cache-poisoning
// fix. A test interrupted by Ctrl-C produces no profile, and the old code
// recorded that as AlwaysRun — a fabricated observation that collectPackage then
// stored under the package fingerprint and the cache served forever, silently
// demoting the package to run-everything.
func TestRunTestRefusesToFabricateCoverageWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	dir := t.TempDir()
	test, unresolved, err := runTest(
		ctx, Options{}, "/nonexistent/test.bin", dir, dir, "TestAnything", newBuilder(), nil)

	require.ErrorIs(t, err, context.Canceled,
		"a cancelled run observed nothing and must say so, not invent an always-run test")
	assert.Empty(t, unresolved)
	assert.False(t, test.AlwaysRun)
	assert.Empty(t, test.Name)
}
