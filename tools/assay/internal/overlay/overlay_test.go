package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteUnderBasenameKeepsTheOriginalName(t *testing.T) {
	workdir := t.TempDir()

	path, err := WriteUnderBasename(workdir, "0", "/repo/pkg/thing_linux_test.go", []byte("package pkg\n"))
	require.NoError(t, err)

	assert.Equal(t, "thing_linux_test.go", filepath.Base(path),
		"the go command reads build constraints off the basename it is handed")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "package pkg\n", string(content))
}

func TestWriteUnderBasenameSeparatesGroups(t *testing.T) {
	workdir := t.TempDir()

	first, err := WriteUnderBasename(workdir, "0", "/a/x.go", []byte("a"))
	require.NoError(t, err)
	second, err := WriteUnderBasename(workdir, "1", "/b/x.go", []byte("b"))
	require.NoError(t, err)

	assert.NotEqual(t, first, second,
		"same basename from different directories must not clobber each other")
}

func TestWriteFileEncodesTheReplaceMap(t *testing.T) {
	workdir := t.TempDir()

	path, err := WriteFile(workdir, map[string]string{
		"/repo/pkg/injected.go": "/tmp/generated.go",
	})
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]map[string]string
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "/tmp/generated.go", decoded["Replace"]["/repo/pkg/injected.go"],
		"a key with no on-disk file is how new files are injected; the shape is the contract")
}

func TestWriteUnderBasenameFailsOnUnwritableWorkdir(t *testing.T) {
	_, err := WriteUnderBasename(filepath.Join(t.TempDir(), "gone", "\x00bad"), "0", "/a/x.go", nil)

	require.Error(t, err)
}
