package gopkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeInspectFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

// TestNameIgnoresConstrainedGenerators pins the fix for the alphabetical
// trap: a //go:build ignore generator declaring package main sorts first under
// exactly the names generators use — build.go, gen.go — and the old
// first-file read injected the harness as package main into a library, failing
// the build on every run.
func TestNameIgnoresConstrainedGenerators(t *testing.T) {
	dir := t.TempDir()
	writeInspectFile(t, dir, "gen.go", "//go:build ignore\n\npackage main\n\nfunc main() {}\n")
	writeInspectFile(t, dir, "lib.go", "package lib\n\nfunc Value() int { return 1 }\n")
	writeInspectFile(t, dir, "lib_test.go",
		"package lib\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {}\n")

	name, err := Name(dir)

	require.NoError(t, err)
	assert.Equal(t, "lib", name, "build constraints decide membership, not sort order")
}

// TestNameServesTestOnlyDirectories pins the fallback: a directory with
// only test files has no production package for go/build to name, and the
// harness still needs a package clause — the test package's, with any _test
// suffix stripped so the injected file lands in the internal test package.
func TestNameServesTestOnlyDirectories(t *testing.T) {
	dir := t.TempDir()
	writeInspectFile(t, dir, "only_test.go",
		"package only_test\n\nimport \"testing\"\n\nfunc TestOnly(t *testing.T) {}\n")

	name, err := Name(dir)

	require.NoError(t, err)
	assert.Equal(t, "only", name)
}

func TestNameFailsOnAnEmptyDirectory(t *testing.T) {
	_, err := Name(t.TempDir())

	assert.Error(t, err)
}
