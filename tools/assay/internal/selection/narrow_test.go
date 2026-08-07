package selection_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/testfixture"
	"github.com/emoss08/assay/internal/vcs"
)

const baseCommit = "0123456789abcdef0123456789abcdef01234567"

type narrowHarness struct {
	root   string
	graph  *graph.Graph
	loader *index.Loader
}

func newNarrowHarness(t *testing.T) narrowHarness {
	t.Helper()
	t.Setenv("GOWORK", "off")

	root := testfixture.Write(t)
	env := testfixture.Env()

	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: root, Env: env})
	require.NoError(t, err)

	store, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	_, err = index.Build(t.Context(), index.Options{
		Root:        root,
		Commit:      baseCommit,
		Graph:       g,
		TreeDigests: testfixture.Digests(t, root),
		Store:       store,
		Env:         env,
		Jobs:        2,
	})
	require.NoError(t, err)

	return narrowHarness{
		root:   root,
		graph:  g,
		loader: index.NewLoader(store, index.NewFingerprinter(g, testfixture.Digests(t, root), nil)),
	}
}

func (h narrowHarness) narrow(t *testing.T, changes []vcs.Change) selection.Narrowed {
	t.Helper()

	result := selection.Select(selection.Options{Graph: h.graph, Changes: changes})

	return selection.Narrow(result, selection.NarrowOptions{
		Graph:      h.graph,
		Loader:     h.loader,
		Changes:    changes,
		BaseCommit: baseCommit,
	})
}

func (h narrowHarness) calcChange(line int) vcs.Change {
	return vcs.Change{
		Path:      filepath.Join(h.root, "calc", "calc.go"),
		Status:    "M",
		Lines:     []vcs.LineRange{{Start: line, End: line}},
		BaseLines: []vcs.LineRange{{Start: line, End: line}},
	}
}

func planFor(t *testing.T, n selection.Narrowed, importPath string) selection.PackagePlan {
	t.Helper()

	for _, plan := range n.Plans {
		if plan.ImportPath == importPath {
			return plan
		}
	}
	t.Fatalf("no plan for %s", importPath)

	return selection.PackagePlan{}
}

func TestNarrowSelectsOnlyCoveringTests(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{h.calcChange(testfixture.CalcAddBodyLine)})

	require.True(t, got.Enabled)
	plan := planFor(t, got, testfixture.Module+"/calc")
	assert.Empty(t, plan.FullReason)
	assert.Equal(t, []string{"TestAdd", "TestSkipped"}, plan.Tests,
		"only Add's test covers the line; the skipped test always runs")
	assert.NotContains(t, plan.Tests, "TestSub")
}

func TestNarrowPicksTheOtherFunction(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{h.calcChange(testfixture.CalcSubBodyLine)})

	plan := planFor(t, got, testfixture.Module+"/calc")
	assert.Equal(t, []string{"TestSkipped", "TestSub"}, plan.Tests)
	assert.NotContains(t, plan.Tests, "TestAdd")
}

func TestNarrowFallsBackOnStructFieldChange(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{h.calcChange(testfixture.CalcStructFieldLine)})

	plan := planFor(t, got, testfixture.Module+"/calc")
	assert.NotEmpty(t, plan.FullReason,
		"a struct field is invisible to coverage, so the whole package must run")
	assert.Contains(t, got.Blocked, filepath.Join(h.root, "calc", "calc.go"))
}

func TestNarrowFallsBackOnSignatureChange(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{h.calcChange(testfixture.CalcSignatureLine)})

	assert.NotEmpty(t, planFor(t, got, testfixture.Module+"/calc").FullReason)
}

func TestNarrowFallsBackOnWholeFileChange(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{{
		Path:   filepath.Join(h.root, "calc", "calc.go"),
		Status: "A",
	}})

	assert.NotEmpty(t, planFor(t, got, testfixture.Module+"/calc").FullReason,
		"an added file has no coverage history")
}

func TestNarrowFallsBackOnNonGoFile(t *testing.T) {
	h := newNarrowHarness(t)

	got := h.narrow(t, []vcs.Change{{
		Path:      filepath.Join(h.root, "svc", "testdata", "seed.sql"),
		Status:    "M",
		Lines:     []vcs.LineRange{{Start: 1, End: 1}},
		BaseLines: []vcs.LineRange{{Start: 1, End: 1}},
	}})

	assert.NotEmpty(t, planFor(t, got, testfixture.Module+"/svc").FullReason,
		"coverage tracks Go lines only")
}

func TestNarrowFallsBackWhenIndexCommitDiffers(t *testing.T) {
	h := newNarrowHarness(t)
	changes := []vcs.Change{h.calcChange(testfixture.CalcAddBodyLine)}

	got := selection.Narrow(
		selection.Select(selection.Options{Graph: h.graph, Changes: changes}),
		selection.NarrowOptions{
			Graph:      h.graph,
			Loader:     h.loader,
			Changes:    changes,
			BaseCommit: "ffffffffffffffffffffffffffffffffffffffff",
		})

	plan := planFor(t, got, testfixture.Module+"/calc")
	assert.Contains(t, plan.FullReason, "index was built at",
		"an index from a different commit has stale line numbers")
}

func TestNarrowFallsBackWithoutBaseCommit(t *testing.T) {
	h := newNarrowHarness(t)
	changes := []vcs.Change{h.calcChange(testfixture.CalcAddBodyLine)}

	got := selection.Narrow(
		selection.Select(selection.Options{Graph: h.graph, Changes: changes}),
		selection.NarrowOptions{Graph: h.graph, Loader: h.loader, Changes: changes})

	assert.False(t, got.Enabled)
	assert.Equal(t, "base commit unknown", got.DisabledReason)
}

func TestNarrowIsDisabledWithoutAnIndex(t *testing.T) {
	h := newNarrowHarness(t)
	changes := []vcs.Change{h.calcChange(testfixture.CalcAddBodyLine)}

	got := selection.Narrow(
		selection.Select(selection.Options{Graph: h.graph, Changes: changes}),
		selection.NarrowOptions{Graph: h.graph, Changes: changes, BaseCommit: baseCommit})

	assert.False(t, got.Enabled)
	assert.Equal(t, "no index available", got.DisabledReason)
	for _, plan := range got.Plans {
		assert.NotEmpty(t, plan.FullReason)
	}
}

func TestNarrowIsDisabledWhenSelectionAlreadyFellBack(t *testing.T) {
	h := newNarrowHarness(t)

	got := selection.Narrow(
		selection.All(h.graph, "--all requested"),
		selection.NarrowOptions{Graph: h.graph, Loader: h.loader, BaseCommit: baseCommit})

	assert.False(t, got.Enabled)
	assert.Contains(t, got.DisabledReason, "already fell back")
}

func TestNarrowFallsBackForPackagesWithoutRecords(t *testing.T) {
	h := newNarrowHarness(t)
	changes := []vcs.Change{h.calcChange(testfixture.CalcAddBodyLine)}

	empty, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	got := selection.Narrow(
		selection.Select(selection.Options{Graph: h.graph, Changes: changes}),
		selection.NarrowOptions{
			Graph:      h.graph,
			Loader:     index.NewLoader(empty, index.NewFingerprinter(h.graph, testfixture.Digests(t, h.root), nil)),
			Changes:    changes,
			BaseCommit: baseCommit,
		})

	assert.Contains(t, planFor(t, got, testfixture.Module+"/calc").FullReason, "no index record")
}

func TestNarrowIgnoresCommentOnlyChanges(t *testing.T) {
	h := newNarrowHarness(t)
	path := filepath.Join(h.root, "calc", "calc.go")

	source, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append([]byte("// leading comment\n"), source...), 0o644))

	got := h.narrow(t, []vcs.Change{{
		Path:      path,
		Status:    "M",
		Lines:     []vcs.LineRange{{Start: 1, End: 1}},
		BaseLines: []vcs.LineRange{{Start: 1, End: 1}},
	}})

	plan := planFor(t, got, testfixture.Module+"/calc")
	assert.Empty(t, plan.FullReason)
	assert.Equal(t, []string{"TestSkipped"}, plan.Tests,
		"a plain comment affects nothing beyond the always-run set")
}

func TestExcludedTestsIsTheComplement(t *testing.T) {
	plan := selection.PackagePlan{
		Tests:    []string{"TestA", "TestC"},
		AllTests: []string{"TestA", "TestB", "TestC", "TestD"},
	}

	assert.Equal(t, []string{"TestB", "TestD"}, plan.ExcludedTests())
}

func TestExcludedTestsIsEmptyForFullPackages(t *testing.T) {
	plan := selection.PackagePlan{
		FullReason: "no index record",
		AllTests:   []string{"TestA"},
	}

	assert.Empty(t, plan.ExcludedTests(), "a package running in full excludes nothing")
}

func TestRunPatternAnchorsAndEscapes(t *testing.T) {
	assert.Equal(t, "^(TestA|TestB)$", selection.RunPattern([]string{"TestA", "TestB"}))
	assert.Equal(t, `^(TestOne\.Two)$`, selection.RunPattern([]string{"TestOne.Two"}))
}
