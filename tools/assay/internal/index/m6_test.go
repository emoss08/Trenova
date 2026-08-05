package index_test

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
)

const m6Module = "example.com/m6"

// The m6 corpus is local to this package rather than grown onto the shared
// testfixture: a deliberately hanging test would slow every consumer of the
// shared fixture, and TestMain packages are only this collector's concern.
//
// plainSource has no init-time execution, so its skipped test must stay
// always-run in both modes. initfulSource executes a var initializer, so its
// skipped test must be attributable in both modes — per-process profiles always
// contained init execution, and the harness path reproduces that by merging the
// init window before the always-run check.
func m6Files(withHangy bool) map[string]string {
	files := map[string]string{
		"go.mod": "module " + m6Module + "\n\ngo 1.26\n",

		"plain/plain.go": `package plain

func Add(a, b int) int {
	sum := a + b

	return sum
}

func Sub(a, b int) int {
	diff := a - b

	return diff
}
`,
		"plain/plain_test.go": `package plain

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad")
	}
}

func TestSub(t *testing.T) {
	if Sub(3, 1) != 2 {
		t.Fatal("bad")
	}
}

func TestSkipped(t *testing.T) {
	t.Skip("never runs")
}
`,

		"initful/initful.go": `package initful

var boot = seed()

func seed() int {
	return 21 * 2
}

func Value() int {
	return boot
}
`,
		"initful/initful_test.go": `package initful

import "testing"

func TestValue(t *testing.T) {
	if Value() != 42 {
		t.Fatal("bad")
	}
}

func TestSkipped(t *testing.T) {
	t.Skip("never runs")
}
`,

		"mainful/mainful.go": `package mainful

func Half(n int) int {
	h := n / 2

	return h
}
`,
		"mainful/mainful_test.go": `package mainful

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestHalf(t *testing.T) {
	if Half(6) != 3 {
		t.Fatal("bad")
	}
}
`,
	}

	if withHangy {
		files["hangy/hangy.go"] = `package hangy

func Fast(n int) int {
	doubled := n * 2

	return doubled
}
`
		files["hangy/hangy_test.go"] = `package hangy

import (
	"testing"
	"time"
)

func TestFast(t *testing.T) {
	if Fast(2) != 4 {
		t.Fatal("bad")
	}
}

func TestHang(t *testing.T) {
	time.Sleep(time.Hour)
}
`
	}

	return files
}

type m6Harness struct {
	root   string
	graph  *graph.Graph
	loader *index.Loader
	stats  index.Stats
}

func m6Setup(t *testing.T, withHangy, legacy bool, timeout time.Duration, packages ...string) m6Harness {
	t.Helper()
	t.Setenv("GOWORK", "off")

	root := t.TempDir()
	digests := make(map[string]string)
	for rel, content := range m6Files(withHangy) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
		digests[rel] = strconv.Itoa(len(content)) + ":" + rel
	}

	env := append(os.Environ(), "GOWORK=off")

	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: root, Env: env})
	require.NoError(t, err)

	store, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	stats, err := index.Build(t.Context(), index.Options{
		Root:        root,
		Commit:      commit,
		Graph:       g,
		TreeDigests: digests,
		Store:       store,
		Env:         env,
		Jobs:        2,
		Timeout:     timeout,
		Packages:    packages,
		Legacy:      legacy,
	})
	require.NoError(t, err)
	require.Empty(t, stats.Failures)

	return m6Harness{
		root:   root,
		graph:  g,
		loader: index.NewLoader(store, index.NewFingerprinter(g, digests, nil)),
		stats:  stats,
	}
}

func (h m6Harness) record(t *testing.T, pkg string) *index.Record {
	t.Helper()

	record, ok := h.loader.Record(m6Module + "/" + pkg)
	require.True(t, ok, "no record for %s", pkg)

	return record
}

func (h m6Harness) mode(t *testing.T, pkg string) string {
	t.Helper()

	for _, outcome := range h.stats.Packages {
		if outcome.Package == m6Module+"/"+pkg {
			return outcome.Mode
		}
	}
	t.Fatalf("no outcome for %s", pkg)

	return ""
}

// normalisedCoverage renders a record's attribution independent of file-interning
// order and duration: per test, the sorted set of resolved file:start-end ranges.
func normalisedCoverage(t *testing.T, record *index.Record) map[string][]string {
	t.Helper()

	out := make(map[string][]string, len(record.Tests))
	for _, test := range record.Tests {
		var ranges []string
		for _, r := range test.Ranges {
			require.Less(t, int(r.FileID), len(record.Files))
			ranges = append(ranges,
				filepath.Base(record.Files[r.FileID])+":"+
					strconv.Itoa(int(r.Start))+"-"+strconv.Itoa(int(r.End)))
		}
		sort.Strings(ranges)
		ranges = dedupe(ranges)
		out[test.Name] = ranges
	}

	return out
}

func dedupe(in []string) []string {
	var out []string
	for i, v := range in {
		if i == 0 || in[i-1] != v {
			out = append(out, v)
		}
	}

	return out
}

// TestSingleProcessAgreesWithLegacy is the test that keeps the fast path honest,
// in the same sense as the schemata agreement test: collecting every test in one
// process is only acceptable if it attributes exactly what one process per test
// attributed.
func TestSingleProcessAgreesWithLegacy(t *testing.T) {
	fast := m6Setup(t, false, false, 0)
	slow := m6Setup(t, false, true, 0)

	for _, pkg := range []string{"plain", "initful", "mainful"} {
		t.Run(pkg, func(t *testing.T) {
			fastRecord := fast.record(t, pkg)
			slowRecord := slow.record(t, pkg)

			assert.Equal(t, slowRecord.TestNames(), fastRecord.TestNames())
			assert.Equal(t, slowRecord.AlwaysRunTests(), fastRecord.AlwaysRunTests(),
				"attributability must not depend on the collection mode")
			assert.Equal(t, slowRecord.Degraded, fastRecord.Degraded)
			assert.Equal(t, normalisedCoverage(t, slowRecord), normalisedCoverage(t, fastRecord),
				"both modes must attribute identical ranges, file for file")

			for _, test := range fastRecord.Tests {
				if !test.AlwaysRun {
					assert.Positive(t, test.Duration, "%s must carry a real duration", test.Name)
				}
			}
		})
	}
}

// TestSingleProcessActuallyTakesTheFastPath stops the agreement test from
// passing for the wrong reason: if every package quietly fell back to
// per-process collection, the two runs would agree perfectly and prove nothing.
func TestSingleProcessActuallyTakesTheFastPath(t *testing.T) {
	h := m6Setup(t, false, false, 0)

	assert.Equal(t, index.CollectionSingleProcess, h.mode(t, "plain"))
	assert.Equal(t, index.CollectionSingleProcess, h.mode(t, "initful"))
	assert.Equal(t, index.CollectionPerProcess, h.mode(t, "mainful"),
		"a package with its own TestMain cannot host the harness")
}

// TestInitWindowKeepsSkippedTestsAttributable pins the parity-critical merge
// order: the init window joins a test's ranges before the empty-check, so a
// skipped test in an init-executing package stays attributable — exactly as its
// per-process profile, which always contained init execution, made it.
func TestInitWindowKeepsSkippedTestsAttributable(t *testing.T) {
	h := m6Setup(t, false, false, 0)

	assert.Contains(t, h.record(t, "plain").AlwaysRunTests(), "TestSkipped",
		"no init execution means a skipped test observed nothing")
	assert.NotContains(t, h.record(t, "initful").AlwaysRunTests(), "TestSkipped",
		"init execution belongs to every test's profile, skipped or not")
}

// TestHangSalvagesCompletedWindows pins the recovery path: a hanging test kills
// the shared process, but every window reported before the stall is complete on
// disk and must be kept; the remainder re-runs per process, restoring the
// per-test timeout semantics for exactly the tests that need them.
func TestHangSalvagesCompletedWindows(t *testing.T) {
	h := m6Setup(t, true, false, 2*time.Second, m6Module+"/hangy")

	assert.Equal(t, index.CollectionSalvaged, h.mode(t, "hangy"))

	record := h.record(t, "hangy")
	assert.Equal(t, []string{"TestFast", "TestHang"}, record.TestNames())

	coverage := normalisedCoverage(t, record)
	assert.NotEmpty(t, coverage["TestFast"],
		"the fast test finished before the hang and its window must survive")

	assert.Contains(t, record.AlwaysRunTests(), "TestHang")
	for _, test := range record.Tests {
		if test.Name == "TestHang" {
			assert.Contains(t, test.Note, "timed out",
				"the salvage re-run must report the hang, not invent an observation")
		}
	}
}

func TestLegacyFlagForcesPerProcessCollection(t *testing.T) {
	h := m6Setup(t, false, true, 0, m6Module+"/plain")

	assert.Equal(t, index.CollectionPerProcess, h.mode(t, "plain"))
}
