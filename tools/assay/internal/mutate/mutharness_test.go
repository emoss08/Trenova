package mutate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/mutate"
)

// initfulSource computes package state at init. This is the harness's one blind
// spot: init runs before TestMain can switch mutants, so an in-process switch
// can never make capacity() misbehave — only a process whose environment
// selected the mutant before init can. The arithmetic mutant in capacity must
// be killed in every mode; a harness without init-site tracking reports it
// survived.
const initfulSource = `package initful

var limit = capacity()

func capacity() int {
	return 40 + 2
}

func Value() int {
	return limit
}
`

const initfulTest = `package initful

import "testing"

func TestValue(t *testing.T) {
	if Value() != 42 {
		t.Fatalf("got %d", Value())
	}
}
`

// panickySource turns a boundary mutant into a crash: `i < len(s)` becoming
// `i <= len(s)` indexes one past the end and panics, which kills the whole
// harness process — the mutants queued behind it must still be judged.
const panickySource = `package panicky

func At(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}

	return -1
}

func Twice(n int) int {
	return n * 2
}

func Sum(a, b int) int {
	return a + b
}
`

const panickyTest = `package panicky

import "testing"

func TestAt(t *testing.T) {
	s := []int{10, 20}
	if got := At(s, 1); got != 20 {
		t.Fatalf("at 1: %d", got)
	}
	if got := At(s, 2); got != -1 {
		t.Fatalf("at 2: %d", got)
	}
}

func TestTwice(t *testing.T) {
	if got := Twice(3); got != 6 {
		t.Fatalf("got %d", got)
	}
}

func TestSum(t *testing.T) {
	if got := Sum(2, 3); got != 5 {
		t.Fatalf("got %d", got)
	}
}
`

// mainfulSource's test package defines its own TestMain, so the harness cannot
// be injected; its mutants must be judged in processes of their own, correctly.
const mainfulSource = `package mainful

func Half(n int) int {
	h := n / 2

	return h
}
`

const mainfulTest = `package mainful

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestHalf(t *testing.T) {
	if got := Half(6); got != 3 {
		t.Fatalf("got %d", got)
	}
}
`

func harnessCorpusFiles() map[string]string {
	return map[string]string{
		"go.mod":                  "module " + fixtureModule + "\n\ngo 1.26\n",
		"initful/initful.go":      initfulSource,
		"initful/initful_test.go": initfulTest,
		"panicky/panicky.go":      panickySource,
		"panicky/panicky_test.go": panickyTest,
		"mainful/mainful.go":      mainfulSource,
		"mainful/mainful_test.go": mainfulTest,
	}
}

// TestHarnessAgreesOnInitExecutedSites pins the taint mechanism: the runtime
// records every site executed during package init, and the harness refuses to
// judge those mutants in-process, handing them to env-selected processes where
// init itself runs mutated. Without that, this package's arithmetic mutant
// would be a false survivor.
func TestHarnessAgreesOnInitExecutedSites(t *testing.T) {
	c := newCorpusFrom(t, harnessCorpusFiles())
	mutants := c.generate(t, "initful")

	harnessResults, err := mutate.Execute(t.Context(), mutants, c.executeOptions())
	require.NoError(t, err)

	reference := c.executeOptions()
	reference.NoSchemata = true
	referenceResults, err := mutate.Execute(t.Context(), mutants, reference)
	require.NoError(t, err)

	requireNoBuildFailures(t, harnessResults)
	requireNoBuildFailures(t, referenceResults)

	assert.Equal(t, outcomesByID(referenceResults), outcomesByID(harnessResults),
		"a site that executes during init must be judged where the environment selects the mutant")

	var killedArithmetic bool
	for _, r := range harnessResults {
		if r.Mutant.Function == "capacity" && r.Mutant.Kind == mutate.KindArithmetic {
			killedArithmetic = r.Outcome.Killed()
			assert.NotEqual(t, mutate.ModeHarness, r.Mode,
				"an init-executed site cannot be judged by an in-process switch")
		}
	}
	assert.True(t, killedArithmetic,
		"40+2 becoming 40-2 changes Value(); surviving means the harness judged init-computed state it cannot reach")
}

// TestHarnessSalvagesACrashingMutant pins the restart path: a mutant that
// panics kills the shared process, its death is its verdict, and the mutants
// queued behind it continue in a fresh process instead of being lost.
func TestHarnessSalvagesACrashingMutant(t *testing.T) {
	c := newCorpusFrom(t, harnessCorpusFiles())

	results, err := mutate.Execute(t.Context(), c.generate(t, "panicky"), c.executeOptions())
	require.NoError(t, err)
	requireNoBuildFailures(t, results)

	var crashKilled bool
	for _, r := range results {
		assert.True(t, r.Outcome.Scored() || r.Outcome == mutate.OutcomeNoCoverage,
			"mutant %s at %s has no verdict: %s", r.Mutant.ID, r.Mutant.Location(), r.Outcome)
		if r.Mutant.Function == "At" && r.Mutant.Kind == mutate.KindBoundary &&
			r.Mutant.Replacement == "<=" {
			crashKilled = r.Outcome.Killed()
		}
	}
	assert.True(t, crashKilled, "indexing one past the end must panic, and the panic is the kill")
}

// TestHarnessFallsBackWhenTestPackageDefinesTestMain pins the structural
// decline: a covering test package with its own TestMain cannot host the
// injected one, and its mutants take env-selected processes — with correct
// verdicts and no spurious fallback warning.
func TestHarnessFallsBackWhenTestPackageDefinesTestMain(t *testing.T) {
	c := newCorpusFrom(t, harnessCorpusFiles())

	results, err := mutate.Execute(t.Context(), c.generate(t, "mainful"), c.executeOptions())
	require.NoError(t, err)
	requireNoBuildFailures(t, results)

	modes := mutate.Modes(results)
	assert.Zero(t, modes[mutate.ModeHarness],
		"a package with its own TestMain cannot host the harness")
	assert.Positive(t, modes[mutate.ModeSchemata])

	for _, r := range results {
		assert.Empty(t, r.Fallback,
			"declining a TestMain package is by design, not a failure to warn about")
		if r.Mutant.Function == "Half" && r.Mutant.Kind == mutate.KindArithmetic {
			assert.True(t, r.Outcome.Killed(), "n/2 becoming n*2 changes Half(6)")
		}
	}
}
