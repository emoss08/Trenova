package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/testfixture"
)

const commit = "0123456789abcdef0123456789abcdef01234567"

type harness struct {
	root    string
	graph   *graph.Graph
	digests map[string]string
	store   *cache.Store
	loader  *index.Loader
}

func setup(t *testing.T) harness {
	t.Helper()
	t.Setenv("GOWORK", "off")

	root := testfixture.Write(t)
	env := testfixture.Env()

	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: root, Env: env})
	require.NoError(t, err)

	store, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	digests := testfixture.Digests(t, root)

	return harness{
		root:    root,
		graph:   g,
		digests: digests,
		store:   store,
		loader:  index.NewLoader(store, index.NewFingerprinter(g, digests, nil)),
	}
}

func (h harness) build(t *testing.T, packages ...string) index.Stats {
	t.Helper()

	stats, err := index.Build(t.Context(), index.Options{
		Root:        h.root,
		Commit:      commit,
		Graph:       h.graph,
		TreeDigests: h.digests,
		Store:       h.store,
		Env:         testfixture.Env(),
		Jobs:        2,
		Packages:    packages,
	})
	require.NoError(t, err)
	require.Empty(t, stats.Failures, "indexing must not fail on the fixture")

	return stats
}

func calcPath(root string) string {
	return filepath.Join(root, "calc", "calc.go")
}

func TestBuildRecordsPerTestCoverage(t *testing.T) {
	h := setup(t)

	h.build(t, testfixture.Module+"/calc")

	record, ok := h.loader.Record(testfixture.Module + "/calc")
	require.True(t, ok, "the calc package must have an index record")

	assert.Equal(t, commit, record.IndexedAt)
	assert.Empty(t, record.Degraded)
	assert.Equal(t, []string{"TestAdd", "TestSkipped", "TestSub"}, record.TestNames())
}

func TestCoverageAttributionIsPerTest(t *testing.T) {
	h := setup(t)
	h.build(t, testfixture.Module+"/calc")

	record, ok := h.loader.Record(testfixture.Module + "/calc")
	require.True(t, ok)

	path := calcPath(h.root)
	require.True(t, record.Knows(path))

	assert.Equal(t, []string{"TestAdd"},
		record.TestsCovering(path, []int{testfixture.CalcAddBodyLine}),
		"only TestAdd executes Add's body")
	assert.Equal(t, []string{"TestSub"},
		record.TestsCovering(path, []int{testfixture.CalcSubBodyLine}),
		"only TestSub executes Sub's body")
}

func TestStructFieldLineHasNoCoverage(t *testing.T) {
	h := setup(t)
	h.build(t, testfixture.Module+"/calc")

	record, ok := h.loader.Record(testfixture.Module + "/calc")
	require.True(t, ok)

	assert.Empty(t, record.TestsCovering(calcPath(h.root), []int{testfixture.CalcStructFieldLine}),
		"declarations never appear in coverage blocks, which is why narrowing must not trust them")
}

func TestSkippedTestIsMarkedAlwaysRun(t *testing.T) {
	h := setup(t)
	h.build(t, testfixture.Module+"/calc")

	record, ok := h.loader.Record(testfixture.Module + "/calc")
	require.True(t, ok)

	assert.Equal(t, []string{"TestSkipped"}, record.AlwaysRunTests(),
		"a test with no attributed statements might skip conditionally, so it always runs")
}

func TestBuildReusesRecordsOnSecondRun(t *testing.T) {
	h := setup(t)

	first := h.build(t, testfixture.Module+"/calc")
	assert.Equal(t, 1, first.Indexed)
	assert.Zero(t, first.Reused)

	second := h.build(t, testfixture.Module+"/calc")
	assert.Zero(t, second.Indexed)
	assert.Equal(t, 1, second.Reused, "an unchanged package must not be re-indexed")
}

func TestBuildReindexesWhenPackageChanges(t *testing.T) {
	h := setup(t)
	h.build(t, testfixture.Module+"/calc")

	require.NoError(t, os.WriteFile(calcPath(h.root),
		[]byte("package calc\n\nfunc Add(a, b int) int {\n\treturn a + b + 0\n}\n\nfunc Sub(a, b int) int {\n\treturn a - b\n}\n"),
		0o644))

	stats := h.reload(t).build(t, testfixture.Module+"/calc")

	assert.Equal(t, 1, stats.Indexed, "changing the package must invalidate its record")
	assert.Zero(t, stats.Reused)
}

func (h harness) reload(t *testing.T) harness {
	t.Helper()

	env := testfixture.Env()
	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: h.root, Env: env})
	require.NoError(t, err)

	digests := testfixture.Digests(t, h.root)

	return harness{
		root:    h.root,
		graph:   g,
		digests: digests,
		store:   h.store,
		loader:  index.NewLoader(h.store, index.NewFingerprinter(g, digests, nil)),
	}
}

func TestBuildIndexesEveryTestablePackageByDefault(t *testing.T) {
	h := setup(t)

	stats := h.build(t)

	assert.Equal(t, len(h.graph.TestablePackages()), stats.Total)
	assert.Positive(t, stats.Tests)
}

func TestFingerprintDiffersPerPackage(t *testing.T) {
	h := setup(t)
	fp := index.NewFingerprinter(h.graph, h.digests, nil)

	calc := fp.For(testfixture.Module + "/calc")
	repo := fp.For(testfixture.Module + "/repo")

	assert.NotEqual(t, calc, repo)
	assert.Equal(t, calc, fp.For(testfixture.Module+"/calc"), "fingerprints must be stable")
}

func TestFingerprintTracksDependencies(t *testing.T) {
	h := setup(t)
	before := index.NewFingerprinter(h.graph, h.digests, nil).For(testfixture.Module + "/svc")

	require.NoError(t, os.WriteFile(filepath.Join(h.root, "repo", "repo.go"),
		[]byte("package repo\n\nfunc Get() int { return 42 }\n"), 0o644))

	after := index.NewFingerprinter(h.graph, testfixture.Digests(t, h.root), nil).
		For(testfixture.Module + "/svc")

	assert.NotEqual(t, before, after,
		"svc's coverage line numbers depend on repo, so a repo edit must invalidate svc")
}

func TestLoaderMissesWhenNothingIndexed(t *testing.T) {
	h := setup(t)

	_, ok := h.loader.Record(testfixture.Module + "/calc")

	assert.False(t, ok)
}

func TestLoaderIsNilSafe(t *testing.T) {
	var loader *index.Loader

	_, ok := loader.Record("anything")

	assert.False(t, ok)
}

func TestBuildStopsOnCancelledContext(t *testing.T) {
	h := setup(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := index.Build(ctx, index.Options{
		Root:        h.root,
		Commit:      commit,
		Graph:       h.graph,
		TreeDigests: h.digests,
		Store:       h.store,
		Env:         testfixture.Env(),
	})

	require.ErrorIs(t, err, context.Canceled,
		"cancelling must abort rather than quietly return partial statistics")
}
