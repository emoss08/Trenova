package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectorySinkCommitsAtomically(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "datasets", "run-1")
	sink, err := newDirectorySink(out)
	require.NoError(t, err)

	file, err := sink.Create("sft-train.jsonl")
	require.NoError(t, err)
	_, err = file.Write([]byte("{}\n"))
	require.NoError(t, err)
	require.NoError(t, file.Close())

	_, err = os.Stat(out)
	require.ErrorIs(t, err, os.ErrNotExist)

	require.NoError(t, sink.commit())
	content, err := os.ReadFile(filepath.Join(out, "sft-train.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(content))

	info, err := os.Stat(filepath.Join(out, "sft-train.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(datasetFileMode), info.Mode().Perm())
}

func TestDirectorySinkRefusesUnsafeNamesAndExistingDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sink, err := newDirectorySink(filepath.Join(root, "out"))
	require.NoError(t, err)
	for _, name := range []string{"", "../escape.jsonl", "nested/file.jsonl", ".hidden"} {
		_, createErr := sink.Create(name)
		require.ErrorIs(t, createErr, errUnsafeDatasetName, name)
	}
	file, err := sink.Create("once.jsonl")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	_, err = sink.Create("once.jsonl")
	require.Error(t, err)

	_, err = newDirectorySink(root)
	require.ErrorContains(t, err, "already exists")
	_, err = newDirectorySink("  ")
	require.Error(t, err)
}

func TestDirectorySinkAbortLeavesNothing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sink, err := newDirectorySink(filepath.Join(root, "out"))
	require.NoError(t, err)
	file, err := sink.Create("sft-train.jsonl")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	sink.abort()
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
