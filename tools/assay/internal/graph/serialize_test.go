package graph

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRoundTripPreservesSelection(t *testing.T) {
	original := buildTestGraph(t)

	payload, err := original.MarshalBinary()
	require.NoError(t, err)

	restored, err := UnmarshalGraph(payload)
	require.NoError(t, err)

	assert.Equal(t, original.Root, restored.Root)
	assert.Equal(t, original.Len(), restored.Len())
	assert.Equal(t, original.TestablePackages(), restored.TestablePackages())
	assert.Equal(t,
		original.AffectedTestPackages([]string{"example.com/repo"}),
		restored.AffectedTestPackages([]string{"example.com/repo"}),
	)
	assert.Equal(t,
		original.AffectedTestPackages([]string{"example.com/fixtures"}),
		restored.AffectedTestPackages([]string{"example.com/fixtures"}),
		"test-only edges must survive serialisation",
	)
}

func TestSnapshotRoundTripPreservesFileAttribution(t *testing.T) {
	original := buildTestGraph(t)

	payload, err := original.MarshalBinary()
	require.NoError(t, err)
	restored, err := UnmarshalGraph(payload)
	require.NoError(t, err)

	path := filepath.Join(testRoot(), "svc", "internal", "util", "testdata", "a.json")

	want, ok := original.PackageForFile(path)
	require.True(t, ok)
	got, ok := restored.PackageForFile(path)
	require.True(t, ok)
	assert.Equal(t, want.ImportPath, got.ImportPath)
}

func TestSnapshotIsStableAcrossEncodes(t *testing.T) {
	g := buildTestGraph(t)

	first, err := g.MarshalBinary()
	require.NoError(t, err)
	second, err := g.MarshalBinary()
	require.NoError(t, err)

	assert.Equal(t, first, second, "map iteration order must not leak into the payload")
}

func TestUnmarshalRejectsGarbage(t *testing.T) {
	_, err := UnmarshalGraph([]byte("not a gob stream"))

	require.Error(t, err)
}

func TestUnmarshalRejectsUnknownVersion(t *testing.T) {
	g := buildTestGraph(t)
	payload, err := g.MarshalBinary()
	require.NoError(t, err)

	restored, err := UnmarshalGraph(payload)
	require.NoError(t, err)
	require.NotNil(t, restored)

	bumped := snapshot{Version: snapshotVersion + 1, Root: g.Root}
	encoded, err := encodeSnapshot(bumped)
	require.NoError(t, err)

	_, err = UnmarshalGraph(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}
