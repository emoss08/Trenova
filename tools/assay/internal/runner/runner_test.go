package runner

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunksSplitsEvenly(t *testing.T) {
	var got [][]string
	for chunk := range chunks([]string{"a", "b", "c", "d", "e"}, 2) {
		got = append(got, slices.Clone(chunk))
	}

	assert.Equal(t, [][]string{{"a", "b"}, {"c", "d"}, {"e"}}, got)
}

func TestChunksHandlesEmptyInput(t *testing.T) {
	var count int
	for range chunks(nil, 4) {
		count++
	}

	assert.Zero(t, count)
}

func TestChunksStopsOnEarlyBreak(t *testing.T) {
	var seen int
	for range chunks([]string{"a", "b", "c", "d"}, 1) {
		seen++

		break
	}

	assert.Equal(t, 1, seen)
}

func TestRunWithNoPackagesIsANoop(t *testing.T) {
	code, err := Run(t.Context(), Options{Root: t.TempDir()})

	require.NoError(t, err)
	assert.Zero(t, code)
}

func writeModule(t *testing.T, passing, failing int) string {
	t.Helper()

	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	write("go.mod", "module example.com/run\n\ngo 1.26\n")
	for i := range passing {
		name := "ok" + strconv.Itoa(i)
		write(name+"/"+name+"_test.go",
			"package "+name+"\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n")
	}
	for i := range failing {
		name := "bad" + strconv.Itoa(i)
		write(name+"/"+name+"_test.go",
			"package "+name+"\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) { t.Fatal(\"boom\") }\n")
	}

	return root
}

func packagePaths(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, "example.com/run/"+prefix+strconv.Itoa(i))
	}

	return out
}

func TestRunReportsSuccess(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 2, 0)

	var out strings.Builder
	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: packagePaths("ok", 2)}},
		Stdout: &out,
		Stderr: io.Discard,
	})

	require.NoError(t, err)
	assert.Zero(t, code)
	assert.Contains(t, out.String(), "ok")
}

func TestRunPropagatesFailureExitCode(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 1)

	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: append(packagePaths("ok", 1), packagePaths("bad", 1)...)}},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})

	require.NoError(t, err, "a failing test suite is a non-zero exit code, not a runner error")
	assert.Equal(t, 1, code)
}

func TestRunSurfacesWorstCodeAcrossChunks(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 1)

	packages := append(packagePaths("ok", 1), packagePaths("bad", 1)...)
	slices.Sort(packages)

	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: packages}},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})

	require.NoError(t, err)
	assert.NotZero(t, code, "a failure in any chunk must surface")
}

func TestRunForwardsExtraArguments(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 0)

	var out strings.Builder
	code, err := Run(t.Context(), Options{
		Root:      root,
		Groups:    []Group{{Packages: packagePaths("ok", 1)}},
		ExtraArgs: []string{"-v", "-count=1"},
		Stdout:    &out,
		Stderr:    io.Discard,
	})

	require.NoError(t, err)
	assert.Zero(t, code)
	assert.Contains(t, out.String(), "--- PASS: TestOK", "-v must reach go test")
}
