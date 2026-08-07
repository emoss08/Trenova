package risk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/risk"
	"github.com/emoss08/assay/internal/vcs"
)

const riskModule = "example.com/riskcorpus"

const riskCommit = "1111111111111111111111111111111111111111"

// The corpus has three shapes of exposure. hot: one tested function beside one
// untested function, in a package with tests. leaf: no tests of its own, but
// its Used function is executed by hot's tests — cross-package coverage the
// analysis must honour — while Unused is executed by nobody. All churn is
// assigned by the test, per scenario.
func riskFiles() map[string]string {
	return map[string]string{
		"go.mod": "module " + riskModule + "\n\ngo 1.26\n",

		"leaf/leaf.go": `package leaf

func Used(n int) int {
	doubled := n * 2

	return doubled
}

func Unused(n int) int {
	tripled := n * 3

	return tripled
}
`,

		"hot/hot.go": `package hot

import "` + riskModule + `/leaf"

func Tested(n int) int {
	total := leaf.Used(n) + 1

	return total
}

func Untested(n int) int {
	shifted := n + 40
	if shifted > 100 {
		return 100
	}

	return shifted
}
`,
		"hot/hot_test.go": `package hot

import "testing"

func TestTested(t *testing.T) {
	if Tested(2) != 5 {
		t.Fatalf("got %d", Tested(2))
	}
}
`,
	}
}

type riskCorpus struct {
	root   string
	graph  *graph.Graph
	loader *index.Loader
}

func newRiskCorpus(t *testing.T) riskCorpus {
	t.Helper()
	t.Setenv("GOWORK", "off")

	root := t.TempDir()
	digests := make(map[string]string)
	for rel, content := range riskFiles() {
		full := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
		digests[rel] = content
	}

	env := append(os.Environ(), "GOWORK=off")
	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: root, Env: env})
	require.NoError(t, err)

	store, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	stats, err := index.Build(t.Context(), index.Options{
		Root:        root,
		Commit:      riskCommit,
		Graph:       g,
		TreeDigests: digests,
		Store:       store,
		Env:         env,
		Jobs:        2,
	})
	require.NoError(t, err)
	require.Empty(t, stats.Failures)

	return riskCorpus{
		root:   root,
		graph:  g,
		loader: index.NewLoader(store, index.NewFingerprinter(g, digests, nil)),
	}
}

func (c riskCorpus) analyze(t *testing.T, churn map[string]vcs.ChurnStat) risk.Report {
	t.Helper()

	report, err := risk.Analyze(risk.Options{
		Root:   c.root,
		Commit: riskCommit,
		Graph:  c.graph,
		Loader: c.loader,
		Churn:  churn,
	})
	require.NoError(t, err)

	return report
}

func find(report risk.Report, function string) (risk.FunctionRisk, bool) {
	for _, fn := range report.Functions {
		if fn.Function == function {
			return fn, true
		}
	}

	return risk.FunctionRisk{}, false
}

// TestRiskRanksChurningUncoveredCodeFirst pins the whole claim: churn times
// uncovered lines, both factors visible, biggest product first.
func TestRiskRanksChurningUncoveredCodeFirst(t *testing.T) {
	c := newRiskCorpus(t)

	report := c.analyze(t, map[string]vcs.ChurnStat{
		"hot/hot.go":   {Commits: 5, LinesChanged: 40},
		"leaf/leaf.go": {Commits: 1, LinesChanged: 3},
	})

	untested, ok := find(report, "Untested")
	require.True(t, ok, "the churning uncovered function is the report's reason to exist")
	assert.Equal(t, 5, untested.Commits)
	assert.Positive(t, untested.UncoveredLines)
	assert.Equal(t, untested.Commits*untested.UncoveredLines, untested.Score)

	unused, ok := find(report, "Unused")
	require.True(t, ok, "a package with no tests of its own is still analyzable")

	require.NotEmpty(t, report.Functions)
	assert.Equal(t, "Untested", report.Functions[0].Function,
		"five commits on uncovered lines outranks one commit on uncovered lines")
	assert.Greater(t, untested.Score, unused.Score)
}

// TestRiskHonoursCrossPackageCoverage pins the union: leaf.Used has no test in
// its own package, but hot's test executes it — reporting it as exposed would
// call tested code untested.
func TestRiskHonoursCrossPackageCoverage(t *testing.T) {
	c := newRiskCorpus(t)

	report := c.analyze(t, map[string]vcs.ChurnStat{
		"leaf/leaf.go": {Commits: 3, LinesChanged: 10},
	})

	_, exposed := find(report, "Used")
	assert.False(t, exposed, "leaf.Used is covered by hot's tests through -coverpkg")

	unused, ok := find(report, "Unused")
	require.True(t, ok)
	assert.Equal(t, 3*unused.UncoveredLines, unused.Score)
}

// TestRiskSeparatesTheQuadrants pins the two muted counters: covered code that
// churns, and uncovered code that does not.
func TestRiskSeparatesTheQuadrants(t *testing.T) {
	c := newRiskCorpus(t)

	report := c.analyze(t, map[string]vcs.ChurnStat{
		"hot/hot.go": {Commits: 2, LinesChanged: 6},
	})

	assert.Positive(t, report.CoveredChurning,
		"hot.Tested churns fully covered; that is reassurance, not risk")
	assert.Positive(t, report.UncoveredStable,
		"leaf.Unused is uncovered but nothing touched its file this window")

	_, exposed := find(report, "Tested")
	assert.False(t, exposed, "a fully covered function is never in the exposed list")
}

// TestRiskRefusesRecordsFromAnotherCommit pins the honesty rule the index
// carries everywhere: line numbers from another commit mean nothing here, so
// such packages count as unindexed rather than being analyzed against them.
func TestRiskRefusesRecordsFromAnotherCommit(t *testing.T) {
	c := newRiskCorpus(t)

	report, err := risk.Analyze(risk.Options{
		Root:   c.root,
		Commit: "2222222222222222222222222222222222222222",
		Graph:  c.graph,
		Loader: c.loader,
		Churn:  nil,
	})
	require.NoError(t, err)

	assert.Zero(t, report.Analyzed)
	assert.Empty(t, report.Functions)
	assert.Positive(t, report.Unindexed)
}
