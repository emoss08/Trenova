package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWatcherSeesRenameBasedSaves pins the atomic-save fix. GoLand's safe write,
// mv, and most codegen replace a file by renaming a temp file over it, which
// reaches fsnotify as a bare Create with no Write — and the old watcher swallowed
// every Create whose path was not a directory. Those editors got zero cycles.
func TestWatcherSeesRenameBasedSaves(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(target, []byte("package a\n"), 0o644))

	w, err := NewWatcher(dir, 100*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go w.Run(ctx)

	staging := filepath.Join(dir, ".a.go.tmp")
	require.NoError(t, os.WriteFile(staging, []byte("package a\n\nvar X = 1\n"), 0o644))
	require.NoError(t, os.Rename(staging, target))

	select {
	case batch := <-w.Batches():
		assert.Contains(t, batch.Paths, target,
			"a rename-based save is one Create event and must still trigger a cycle")
	case <-time.After(5 * time.Second):
		t.Fatal("the rename-based save produced no batch")
	}
}

// TestStickyFailureSurvivesANarrowedPass pins the false-green fix: a run that
// executed only a subset of a package's tests may clear exactly those names,
// never the whole entry.
func TestStickyFailureSurvivesANarrowedPass(t *testing.T) {
	failing := map[string]*failure{
		"example.com/a": {tests: map[string]struct{}{"TestBroken": {}}},
	}

	recordPass(failing, RunResult{
		ImportPath: "example.com/a",
		Tests:      []string{"TestOther"},
		Passed:     true,
	})

	require.Contains(t, failing, "example.com/a",
		"a pass that never executed TestBroken cannot declare it fixed")
	assert.Contains(t, failing["example.com/a"].tests, "TestBroken")

	recordPass(failing, RunResult{
		ImportPath: "example.com/a",
		Tests:      []string{"TestBroken"},
		Passed:     true,
	})
	assert.NotContains(t, failing, "example.com/a",
		"executing the broken test and passing is what clears it")
}

func TestWholePackageFailureNeedsAFullPassToClear(t *testing.T) {
	failing := make(map[string]*failure)

	recordFailure(failing, RunResult{ImportPath: "example.com/a", Output: []byte("panic: boom\n")})
	require.True(t, failing["example.com/a"].wholePackage,
		"a failure that names no test can only be cleared by a full run")

	recordPass(failing, RunResult{ImportPath: "example.com/a", Tests: []string{"TestOne"}, Passed: true})
	assert.Contains(t, failing, "example.com/a")

	recordPass(failing, RunResult{ImportPath: "example.com/a", Passed: true})
	assert.NotContains(t, failing, "example.com/a")
}

func TestFailureNamesUnionRatherThanOverwrite(t *testing.T) {
	failing := make(map[string]*failure)

	recordFailure(failing, RunResult{
		ImportPath: "example.com/a",
		Tests:      []string{"TestOne"},
		Output:     []byte("--- FAIL: TestOne (0.00s)\n"),
	})
	recordFailure(failing, RunResult{
		ImportPath: "example.com/a",
		Tests:      []string{"TestTwo"},
		Output:     []byte("--- FAIL: TestTwo (0.00s)\n"),
	})

	entry := failing["example.com/a"]
	assert.Contains(t, entry.tests, "TestOne",
		"a second failing test must not erase the memory of the first")
	assert.Contains(t, entry.tests, "TestTwo")
}

func TestUnjudgedRunsKeepStickiness(t *testing.T) {
	failing := map[string]*failure{
		"example.com/a": {tests: map[string]struct{}{"TestBroken": {}}},
	}

	recordUnjudged(failing, RunResult{ImportPath: "example.com/a"})
	assert.Contains(t, failing["example.com/a"].tests, "TestBroken",
		"a build error is not evidence the test passed")

	recordUnjudged(failing, RunResult{ImportPath: "example.com/b"})
	assert.True(t, failing["example.com/b"].wholePackage,
		"a hang or build error on a green package must be re-checked in full")
}

func TestMergeFailureWidensANarrowedRun(t *testing.T) {
	run := PlannedRun{Pattern: "^(TestOther)$", Tests: []string{"TestOther"}}
	mergeFailureIntoRun(&run, &failure{tests: map[string]struct{}{"TestBroken": {}}})

	assert.Equal(t, []string{"TestBroken", "TestOther"}, run.Tests,
		"the failing test rides along with whatever the cycle planned")

	full := PlannedRun{Pattern: "^(TestOther)$", Tests: []string{"TestOther"}}
	mergeFailureIntoRun(&full, &failure{wholePackage: true})
	assert.Empty(t, full.Pattern, "a whole-package failure forces the whole package")
}

// TestImportSignatureTracksBuildConstraints pins the graph-staleness fix: a
// //go:build line is a comment to the parser but not to the graph.
func TestImportSignatureTracksBuildConstraints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.go")

	require.NoError(t, os.WriteFile(path,
		[]byte("//go:build ignore\n\npackage p\n\nimport \"os\"\n\nvar _ = os.Args\n"), 0o644))
	constrained, err := importSignature(path)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path,
		[]byte("package p\n\nimport \"os\"\n\nvar _ = os.Args\n"), 0o644))
	unconstrained, err := importSignature(path)
	require.NoError(t, err)

	assert.NotEqual(t, constrained, unconstrained,
		"removing a build constraint changes package membership and must reload the graph")
}

func TestBinaryNamesAreInjective(t *testing.T) {
	assert.NotEqual(t, binaryName("example.com/a/b"), binaryName("example.com/a_b"),
		"colliding names would let one package execute another's binary")
	assert.NotEqual(t, binaryName("example.com/foo/bar"), binaryName("example.com/foo_bar"))
}

// TestBenchmarkEditRunsTheWholePackage pins the false-green fix: -test.run does
// not execute benchmarks, so narrowing to one would report a green cycle in
// which nothing ran.
func TestBenchmarkEditRunsTheWholePackage(t *testing.T) {
	r := newRepo(t)
	r.write("alpha/alpha_test.go", alphaTest+
		"\nfunc BenchmarkDouble(b *testing.B) {\n\tfor range b.N {\n\t\tDouble(2)\n\t}\n}\n")
	r.git("add", ".")
	r.git("commit", "-qm", "add benchmark")

	r.write("alpha/alpha_test.go", strings.Replace(alphaTest+
		"\nfunc BenchmarkDouble(b *testing.B) {\n\tfor range b.N {\n\t\tDouble(2)\n\t}\n}\n",
		"Double(2)\n", "Double(3)\n", 1))

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("alpha/alpha_test.go")})
	require.NoError(t, err)

	require.Len(t, cycle.Runs, 1)
	assert.Empty(t, cycle.Runs[0].Pattern,
		"an edit inside a benchmark cannot narrow to it; the package runs in full")
}

// TestDirtyDocsDoNotDisableRefinement pins the monorepo fix: an unattributable
// non-Go file elsewhere in the repository must not switch off the
// one-test-refinement everywhere.
func TestDirtyDocsDoNotDisableRefinement(t *testing.T) {
	r := newRepo(t)
	r.write("docs/notes.md", "# scratch\n")
	r.write("alpha/alpha_test.go", strings.Replace(alphaTest, "!= 4", "!= 2+2", 1))

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("alpha/alpha_test.go")})
	require.NoError(t, err)

	require.Len(t, cycle.Runs, 1)
	assert.Equal(t, []string{"TestDouble"}, cycle.Runs[0].Tests,
		"a dirty markdown file is not a reason to run the whole package")
}

func TestResyncBatchInvalidatesTheMemoisedGraph(t *testing.T) {
	r := newRepo(t)
	p := planner(r)

	_, err := p.Plan(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, p.memoGraph)

	p.Invalidate()
	assert.Nil(t, p.memoGraph)
	assert.Nil(t, p.signatures)
}
