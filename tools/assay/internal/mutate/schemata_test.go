package mutate_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/mutate"
)

// kindsSource fires every mutator, and includes the named types the schemata
// rewrite has to decline, so a single run covers both paths at once.
const kindsSource = `package kinds

type Meter int

type Flag bool

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}

func Mul(a, b int) int {
	return a * b
}

func Div(a, b int) int {
	return a / b
}

func Mod(a, b int) int {
	return a % b
}

func Below(a, b int) bool {
	return a < b
}

func AtMost(a, b int) bool {
	return a <= b
}

func Above(a, b int) bool {
	return a > b
}

func AtLeast(a, b int) bool {
	return a >= b
}

func Same(a, b string) bool {
	return a == b
}

func Differs(a, b string) bool {
	return a != b
}

func Both(a, b bool) bool {
	return a && b
}

func Either(a, b bool) bool {
	return a || b
}

func Yes() bool {
	return true
}

func No() bool {
	return false
}

func Count(n int) int {
	total := 0
	for range n {
		total++
	}

	return total
}

func Pick(flag bool, a, b int) int {
	if flag {
		return a
	}

	return b
}

func Span(d Meter, n int) Meter {
	return d * Meter(n)
}

func NamedConnector(a, b Flag) Flag {
	return a && b
}

func NamedResult(a, b int) Flag {
	return a < b
}
`

const kindsTest = `package kinds

import "testing"

func TestArithmetic(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Fatalf("add %d", got)
	}
	if got := Sub(5, 3); got != 2 {
		t.Fatalf("sub %d", got)
	}
	if got := Mul(6, 3); got != 18 {
		t.Fatalf("mul %d", got)
	}
	if got := Div(6, 3); got != 2 {
		t.Fatalf("div %d", got)
	}
	if got := Mod(7, 3); got != 1 {
		t.Fatalf("mod %d", got)
	}
}

func TestBoundaries(t *testing.T) {
	if Below(5, 5) {
		t.Fatal("5 is not below 5")
	}
	if !AtMost(5, 5) {
		t.Fatal("5 is at most 5")
	}
	if Above(5, 5) {
		t.Fatal("5 is not above 5")
	}
	if !AtLeast(5, 5) {
		t.Fatal("5 is at least 5")
	}
}

func TestEquality(t *testing.T) {
	if !Same("a", "a") {
		t.Fatal("a is a")
	}
	if !Differs("a", "b") {
		t.Fatal("a differs from b")
	}
}

func TestConnectors(t *testing.T) {
	if Both(true, false) {
		t.Fatal("true and false is false")
	}
	if !Either(false, true) {
		t.Fatal("false or true is true")
	}
	if NamedConnector(true, false) {
		t.Fatal("named and")
	}
	if NamedResult(5, 5) {
		t.Fatal("5 is not below 5")
	}
}

func TestLiterals(t *testing.T) {
	if !Yes() {
		t.Fatal("yes")
	}
	if No() {
		t.Fatal("no")
	}
}

func TestCount(t *testing.T) {
	if got := Count(3); got != 3 {
		t.Fatalf("count %d", got)
	}
}

func TestPick(t *testing.T) {
	if got := Pick(true, 1, 2); got != 1 {
		t.Fatalf("pick true %d", got)
	}
	if got := Pick(false, 1, 2); got != 2 {
		t.Fatalf("pick false %d", got)
	}
}

func TestSpan(t *testing.T) {
	if got := Span(2, 3); got != 6 {
		t.Fatalf("span %d", got)
	}
}
`

// dataSource needs its working directory to be the package directory, exactly as
// `go test` arranges. The int it returns is unasserted, so the mutation there must
// survive — and it can only survive if the test itself passed for the right reason.
const dataSource = `package data

import "os"

func Load() (string, int) {
	raw, err := os.ReadFile("testdata/value.txt")
	if err != nil {
		return "", 0
	}

	return string(raw), len(raw) + 1
}
`

const dataTest = `package data

import "testing"

func TestLoad(t *testing.T) {
	got, _ := Load()
	if got != "seven" {
		t.Fatalf("got %q", got)
	}
}
`

func schemataCorpusFiles() map[string]string {
	return map[string]string{
		"go.mod":                  "module " + fixtureModule + "\n\ngo 1.26\n",
		"kinds/kinds.go":          kindsSource,
		"kinds/kinds_test.go":     kindsTest,
		"data/data.go":            dataSource,
		"data/data_test.go":       dataTest,
		"data/testdata/value.txt": "seven",
	}
}

func outcomesByID(results []mutate.Result) map[string]mutate.Outcome {
	out := make(map[string]mutate.Outcome, len(results))
	for _, r := range results {
		out[r.Mutant.ID] = r.Outcome
	}

	return out
}

func survivorIDs(results []mutate.Result) []string {
	var out []string
	for _, r := range results {
		if r.Outcome == mutate.OutcomeSurvived {
			out = append(out, r.Mutant.ID+" "+r.Mutant.Function+" "+
				r.Mutant.Original+"->"+r.Mutant.Replacement)
		}
	}
	sort.Strings(out)

	return out
}

// TestSchemataAgreesWithOverlay is the test that keeps every fast path honest.
//
// The three modes trade compiles and processes for speed: overlay pays a compile
// per mutant, per-process schemata shares the compile, and the harness shares the
// processes too. None of it is worth anything unless all three reach identical
// verdicts. Any divergence — a rewrite that changed evaluation order, a switch
// that leaked into a neighbouring site, an init that ran under the wrong mutant —
// shows up here as a mutant the modes disagree about.
func TestSchemataAgreesWithOverlay(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	for _, pkg := range []string{"kinds", "data"} {
		t.Run(pkg, func(t *testing.T) {
			mutants := c.generate(t, pkg)

			harness := c.executeOptions()
			harnessResults, err := mutate.Execute(t.Context(), mutants, harness)
			require.NoError(t, err)

			perProcess := c.executeOptions()
			perProcess.NoHarness = true
			perProcessResults, err := mutate.Execute(t.Context(), mutants, perProcess)
			require.NoError(t, err)

			separate := c.executeOptions()
			separate.NoSchemata = true
			separateResults, err := mutate.Execute(t.Context(), mutants, separate)
			require.NoError(t, err)

			requireNoBuildFailures(t, harnessResults)
			requireNoBuildFailures(t, perProcessResults)
			requireNoBuildFailures(t, separateResults)

			assert.Equal(t, outcomesByID(separateResults), outcomesByID(perProcessResults),
				"a shared binary must reach the same verdict as one binary per mutant")
			assert.Equal(t, outcomesByID(separateResults), outcomesByID(harnessResults),
				"switching mutants inside one process must reach the same verdict as a process per mutant")
			assert.Equal(t, survivorIDs(separateResults), survivorIDs(perProcessResults))
			assert.Equal(t, survivorIDs(separateResults), survivorIDs(harnessResults))
		})
	}
}

// TestSchemataActuallyTakesTheFastPath stops the agreement test above from passing
// for the wrong reason. If every mutant quietly fell back to its own binary or its
// own process the verdicts would match perfectly and the milestone would be
// worthless.
func TestSchemataActuallyTakesTheFastPath(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	results, err := mutate.Execute(t.Context(), c.generate(t, "kinds"), c.executeOptions())
	require.NoError(t, err)

	modes := mutate.Modes(results)
	assert.Positive(t, modes[mutate.ModeHarness],
		"the harness must actually judge mutants, or its agreement proves nothing")
	assert.Zero(t, modes[mutate.ModeSchemata],
		"nothing in this package runs at init and no test package defines TestMain, so no mutant needs a process of its own")
	assert.Greater(t, modes[mutate.ModeHarness], modes[mutate.ModeOverlay],
		"most of this package is plain int and bool; it should mostly share a binary")
	assert.Positive(t, modes[mutate.ModeOverlay],
		"the named types in this package cannot be rewritten, so both paths must run")
}

func TestNoHarnessForcesAProcessPerMutant(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	opts := c.executeOptions()
	opts.NoHarness = true

	results, err := mutate.Execute(t.Context(), c.generate(t, "kinds"), opts)
	require.NoError(t, err)

	modes := mutate.Modes(results)
	assert.Zero(t, modes[mutate.ModeHarness])
	assert.Positive(t, modes[mutate.ModeSchemata])
}

func TestNoSchemataForcesOneBinaryPerMutant(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	opts := c.executeOptions()
	opts.NoSchemata = true

	results, err := mutate.Execute(t.Context(), c.generate(t, "kinds"), opts)
	require.NoError(t, err)

	modes := mutate.Modes(results)
	assert.Zero(t, modes[mutate.ModeSchemata])
	assert.Positive(t, modes[mutate.ModeOverlay])
}

// TestTestBinaryRunsInThePackageDirectory pins a bug the schemata work uncovered:
// mutation ran compiled test binaries from the workspace root, so any test opening
// testdata/ failed for reasons that had nothing to do with the mutant — and every
// mutant it judged was recorded as killed.
func TestTestBinaryRunsInThePackageDirectory(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	results, err := mutate.Execute(t.Context(), c.generate(t, "data"), c.executeOptions())
	require.NoError(t, err)

	requireNoBuildFailures(t, results)

	var found bool
	for _, r := range results {
		if r.Mutant.Kind != mutate.KindArithmetic {
			continue
		}
		found = true
		assert.Equal(t, mutate.OutcomeSurvived, r.Outcome,
			"nothing asserts the length Load returns, so this mutation cannot be killed; "+
				"reporting it as killed means the test failed for the wrong reason")
	}
	require.True(t, found, "the length arithmetic in Load must produce a mutant")
}

func TestSchemataRuntimeIsNotWrittenToTheSourceTree(t *testing.T) {
	c := newCorpusFrom(t, schemataCorpusFiles())

	_, err := mutate.Execute(t.Context(), c.generate(t, "kinds"), c.executeOptions())
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(c.root, "kinds", mutate.SchemataFileName))
	assert.True(t, os.IsNotExist(statErr),
		"the injected runtime reaches the compiler through -overlay and must never touch the tree")
}
