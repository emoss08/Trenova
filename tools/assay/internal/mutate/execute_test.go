package mutate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/mutate"
)

const corpusCommit = "abcdef0123456789abcdef0123456789abcdef01"

// strongSource is fully asserted: every mutation should change an observable
// result, so the whole suite should reach MSI 100.
const strongSource = `package strong

func AtCapacity(n, limit int) bool {
	if n > limit {
		return true
	}

	return false
}

func Sum(a, b int) int {
	return a + b
}
`

const strongTest = `package strong

import "testing"

func TestBelowCapacity(t *testing.T) {
	if AtCapacity(3, 10) {
		t.Fatal("3 is not over 10")
	}
}

func TestExactlyAtCapacity(t *testing.T) {
	if AtCapacity(10, 10) {
		t.Fatal("10 is not over 10")
	}
}

func TestOverCapacity(t *testing.T) {
	if !AtCapacity(11, 10) {
		t.Fatal("11 is over 10")
	}
}

func TestSum(t *testing.T) {
	if got := Sum(2, 3); got != 5 {
		t.Fatalf("got %d", got)
	}
}
`

// equivalentSource has a boundary no test can pin down: at n == limit both the
// original `>` and the mutated `>=` return limit, because limit and n are equal
// there. The surviving mutant is semantically identical to the original, not a
// hole in the suite - which is the equivalent-mutant problem in miniature.
const equivalentSource = `package equivalent

func Clamp(n, limit int) int {
	if n > limit {
		return limit
	}

	return n
}
`

const equivalentTest = `package equivalent

import "testing"

func TestClampBelowLimit(t *testing.T) {
	if got := Clamp(3, 10); got != 3 {
		t.Fatalf("got %d", got)
	}
}

func TestClampAtLimit(t *testing.T) {
	if got := Clamp(10, 10); got != 10 {
		t.Fatalf("got %d", got)
	}
}

func TestClampAboveLimit(t *testing.T) {
	if got := Clamp(11, 10); got != 10 {
		t.Fatalf("got %d", got)
	}
}
`

// weakSource is exercised but barely asserted: the test calls Discount and only
// checks the result is non-negative, so arithmetic mutations survive.
const weakSource = `package weak

func Discount(price, off int) int {
	result := price - off
	if result < 0 {
		return 0
	}

	return result
}
`

const weakTest = `package weak

import "testing"

func TestDiscountIsNotNegative(t *testing.T) {
	if Discount(100, 10) < 0 {
		t.Fatal("negative")
	}
}
`

// loopSource mutates into an infinite loop: forcing the guard false means the
// counter never terminates, which must be reported as killed by timeout.
const loopSource = `package loop

func CountTo(limit int) int {
	total := 0
	for {
		if total >= limit {
			break
		}
		total++
	}

	return total
}
`

const loopTest = `package loop

import "testing"

func TestCountTo(t *testing.T) {
	if got := CountTo(3); got != 3 {
		t.Fatalf("got %d", got)
	}
}
`

type corpus struct {
	root   string
	graph  *graph.Graph
	loader *index.Loader
	scope  *mutate.Scope
}

func corpusFiles() map[string]string {
	return map[string]string{
		"go.mod":                        "module " + fixtureModule + "\n\ngo 1.26\n",
		"strong/strong.go":              strongSource,
		"strong/strong_test.go":         strongTest,
		"equivalent/equivalent.go":      equivalentSource,
		"equivalent/equivalent_test.go": equivalentTest,
		"weak/weak.go":                  weakSource,
		"weak/weak_test.go":             weakTest,
		"loop/loop.go":                  loopSource,
		"loop/loop_test.go":             loopTest,
	}
}

func writeCorpus(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()

	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	return root
}

func digestsOf(t *testing.T, root string) map[string]string {
	t.Helper()

	digests := make(map[string]string)
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		digests[filepath.ToSlash(rel)] = string(content)

		return nil
	}))

	return digests
}

func newCorpus(t *testing.T) corpus {
	t.Helper()

	return newCorpusFrom(t, corpusFiles())
}

func newCorpusFrom(t *testing.T, files map[string]string) corpus {
	t.Helper()
	t.Setenv("GOWORK", "off")

	root := writeCorpus(t, files)
	env := append(os.Environ(), "GOWORK=off")

	g, err := graph.Load(t.Context(), graph.LoadOptions{Root: root, Env: env})
	require.NoError(t, err)

	store, err := cache.NewStore(t.TempDir())
	require.NoError(t, err)

	digests := digestsOf(t, root)

	stats, err := index.Build(t.Context(), index.Options{
		Root:        root,
		Commit:      corpusCommit,
		Graph:       g,
		TreeDigests: digests,
		Store:       store,
		Env:         env,
		Jobs:        2,
	})
	require.NoError(t, err)
	require.Empty(t, stats.Failures)

	loader := index.NewLoader(store, index.NewFingerprinter(g, digests, nil))

	scope, err := mutate.NewScope(mutate.ScopeOptions{
		Graph:      g,
		Loader:     loader,
		BaseCommit: corpusCommit,
	})
	require.NoError(t, err)

	return corpus{root: root, graph: g, loader: loader, scope: scope}
}

func (c corpus) generate(t *testing.T, pkg string) []mutate.Mutant {
	t.Helper()

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/" + pkg,
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests:    c.scope.Tests,
	})
	require.NoError(t, err)
	require.NotEmpty(t, mutants, "the corpus must produce mutants for %s", pkg)

	return mutants
}

func (c corpus) executeOptions() mutate.ExecuteOptions {
	return mutate.ExecuteOptions{
		Root:       c.root,
		Env:        append(os.Environ(), "GOWORK=off"),
		Jobs:       2,
		Budget:     c.scope.Budget,
		MinTimeout: 3 * time.Second,
		PackageDir: func(importPath string) string {
			pkg, ok := c.graph.Package(importPath)
			if !ok {
				return ""
			}

			return pkg.Dir
		},
	}
}

func (c corpus) run(t *testing.T, pkg string) ([]mutate.Result, mutate.Score) {
	t.Helper()

	results, err := mutate.Execute(t.Context(), c.generate(t, pkg), c.executeOptions())
	require.NoError(t, err)

	return results, mutate.Evaluate(results, mutate.NewBaseline())
}

func requireNoBuildFailures(t *testing.T, results []mutate.Result) {
	t.Helper()

	for _, r := range results {
		require.NotEqual(t, mutate.OutcomeNotBuilt, r.Outcome,
			"mutant %s at %s failed to build: %s", r.Mutant.ID, r.Mutant.Location(), r.Detail)
	}
}

func TestStrongSuiteKillsEveryMutant(t *testing.T) {
	c := newCorpus(t)

	results, score := c.run(t, "strong")

	requireNoBuildFailures(t, results)
	assert.Zero(t, score.Survived, "a fully asserted suite should leave no survivors")
	assert.Positive(t, score.Killed)
	assert.InDelta(t, 100.0, score.MSI(), 0.001)
}

func TestEquivalentMutantSurvivesAThoroughSuite(t *testing.T) {
	c := newCorpus(t)

	results, score := c.run(t, "equivalent")

	requireNoBuildFailures(t, results)
	require.Len(t, score.NewSurvivors, 1,
		"exactly the boundary mutant should survive, and not because the suite is weak")

	survivor := score.NewSurvivors[0].Mutant
	assert.Equal(t, mutate.KindBoundary, survivor.Kind)
	assert.Equal(t, ">", survivor.Original)
	assert.Equal(t, ">=", survivor.Replacement)
	assert.Equal(t, "Clamp", survivor.Function)
}

func TestWeakSuiteLeavesTheExpectedSurvivors(t *testing.T) {
	c := newCorpus(t)

	results, score := c.run(t, "weak")

	requireNoBuildFailures(t, results)
	require.Positive(t, score.Survived, "asserting only non-negativity cannot catch arithmetic changes")

	var kinds []mutate.Kind
	for _, r := range score.NewSurvivors {
		kinds = append(kinds, r.Mutant.Kind)
		assert.Equal(t, "Discount", r.Mutant.Function)
	}
	assert.Contains(t, kinds, mutate.KindArithmetic,
		"price - off becoming price + off is exactly the gap this suite has")
	assert.Less(t, score.MSI(), 100.0)
}

func TestInfiniteLoopMutantIsKilledByTimeout(t *testing.T) {
	c := newCorpus(t)

	results, _ := c.run(t, "loop")

	requireNoBuildFailures(t, results)

	var timeouts int
	for _, r := range results {
		if r.Outcome == mutate.OutcomeTimeout {
			timeouts++
			assert.Contains(t, r.Detail, "exceeded")
		}
	}
	assert.Positive(t, timeouts, "forcing the loop guard false must time out rather than hang the run")
}

func TestKilledMutantNamesTheFailingTest(t *testing.T) {
	c := newCorpus(t)

	results, _ := c.run(t, "strong")

	var named int
	for _, r := range results {
		if r.Outcome == mutate.OutcomeKilled {
			assert.NotEmpty(t, r.KilledBy, "a kill should say which package caught it")
			if r.Detail != "" {
				named++
			}
		}
	}
	assert.Positive(t, named)
}

func TestOnlyCoveringTestsAreRun(t *testing.T) {
	c := newCorpus(t)

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests:    c.scope.Tests,
	})
	require.NoError(t, err)

	for _, m := range mutants {
		if m.Function != "Sum" {
			continue
		}
		plan := m.Tests()
		require.NotEmpty(t, plan)
		for _, tests := range plan {
			assert.Equal(t, []string{"TestSum"}, tests,
				"a mutation in Sum should only run the test that covers Sum")
		}

		return
	}
	t.Fatal("no mutant found in Sum")
}

func TestBaselineSuppressesKnownSurvivors(t *testing.T) {
	c := newCorpus(t)
	results, fresh := c.run(t, "weak")

	require.NotEmpty(t, fresh.NewSurvivors)

	baseline := mutate.FromResults(results)
	assert.Equal(t, fresh.Survived, baseline.Len())

	suppressed := mutate.Evaluate(results, baseline)
	assert.Empty(t, suppressed.NewSurvivors, "a recorded survivor must not be reported again")
	assert.Equal(t, fresh.Survived, suppressed.Accepted)
	assert.InDelta(t, fresh.MSI(), suppressed.MSI(), 0.001,
		"the baseline suppresses reporting, it does not inflate the score")
}

func TestBaselineDoesNotSuppressANewSurvivor(t *testing.T) {
	c := newCorpus(t)
	results, _ := c.run(t, "weak")

	baseline := mutate.FromResults(results)

	extra := append([]mutate.Result(nil), results...)
	for i := range extra {
		if extra[i].Outcome == mutate.OutcomeSurvived {
			extra[i].Mutant.ID = "0000000000000000"

			break
		}
	}

	score := mutate.Evaluate(extra, baseline)
	assert.Len(t, score.NewSurvivors, 1, "an unrecognised survivor must fail the run")
}

func TestScoreExcludesUnbuiltAndUncoveredMutants(t *testing.T) {
	score := mutate.Evaluate([]mutate.Result{
		{Outcome: mutate.OutcomeKilled},
		{Outcome: mutate.OutcomeSurvived},
		{Outcome: mutate.OutcomeNotBuilt},
		{Outcome: mutate.OutcomeNoCoverage},
	}, mutate.NewBaseline())

	assert.Equal(t, 2, score.Scored())
	assert.InDelta(t, 50.0, score.MSI(), 0.001,
		"a mutant assay could not build is an assay defect, and an uncovered line is already a known gap")
}

func TestTimeoutCountsAsKilled(t *testing.T) {
	score := mutate.Evaluate([]mutate.Result{
		{Outcome: mutate.OutcomeTimeout},
		{Outcome: mutate.OutcomeSurvived},
	}, mutate.NewBaseline())

	assert.InDelta(t, 50.0, score.MSI(), 0.001)
	assert.True(t, mutate.OutcomeTimeout.Killed())
}

func TestScopeRequiresAnIndex(t *testing.T) {
	_, err := mutate.NewScope(mutate.ScopeOptions{BaseCommit: corpusCommit})

	require.Error(t, err)
}

func TestScopeRequiresABaseCommit(t *testing.T) {
	c := newCorpus(t)

	_, err := mutate.NewScope(mutate.ScopeOptions{Graph: c.graph, Loader: c.loader})

	require.Error(t, err, "an index from another commit has line numbers that mean nothing here")
}

const redSource = `package red

func Add(a, b int) int {
	return a + b
}
`

// redTest already fails, which is the point: a mutant judged by it would be
// recorded as killed no matter what the mutation did.
const redTest = `package red

import "testing"

func TestAddIsBroken(t *testing.T) {
	if Add(1, 1) != 3 {
		t.Fatal("this assertion is simply wrong")
	}
}
`

func writeRedPackage(t *testing.T, root string) {
	t.Helper()

	dir := filepath.Join(root, "red")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "red.go"), []byte(redSource), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "red_test.go"), []byte(redTest), 0o644))
}

func TestPreflightDetectsAlreadyFailingTests(t *testing.T) {
	c := newCorpus(t)
	writeRedPackage(t, c.root)

	scope, err := mutate.NewScope(mutate.ScopeOptions{
		Graph:      c.graph,
		Loader:     c.loader,
		BaseCommit: corpusCommit,
	})
	require.NoError(t, err)

	mutants := []mutate.Mutant{}
	generated, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: scope.Eligible,
		Tests: func(string, int) mutate.TestPlan {
			return mutate.TestPlan{fixtureModule + "/red": {"TestAddIsBroken"}}
		},
	})
	require.NoError(t, err)
	mutants = append(mutants, generated...)
	require.NotEmpty(t, mutants)

	opts := mutate.ExecuteOptions{Root: c.root, Env: append(os.Environ(), "GOWORK=off")}
	pre, err := mutate.RunPreflight(t.Context(), mutants, opts)
	require.NoError(t, err)

	assert.False(t, pre.Clean(), "an already-failing covering test must be reported")
	assert.Contains(t, pre.Names(), fixtureModule+"/red.TestAddIsBroken")
}

func TestPreflightIsCleanForAGreenSuite(t *testing.T) {
	c := newCorpus(t)

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests:    c.scope.Tests,
	})
	require.NoError(t, err)

	pre, err := mutate.RunPreflight(t.Context(), mutants,
		mutate.ExecuteOptions{Root: c.root, Env: append(os.Environ(), "GOWORK=off")})
	require.NoError(t, err)

	assert.True(t, pre.Clean())
	assert.Empty(t, pre.Names())
}

func TestExcludeDropsFailingTestsFromPlans(t *testing.T) {
	c := newCorpus(t)

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests:    c.scope.Tests,
	})
	require.NoError(t, err)
	require.NotEmpty(t, mutants)

	pkg := fixtureModule + "/strong"
	filtered := mutate.Exclude(mutants, map[string][]string{pkg: {"TestSum"}})

	for _, m := range filtered {
		for _, tests := range m.Tests() {
			assert.NotContains(t, tests, "TestSum")
		}
	}
}

func TestExcludeLeavesMutantsWithNoTestsAsUncovered(t *testing.T) {
	c := newCorpus(t)

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests: func(string, int) mutate.TestPlan {
			return mutate.TestPlan{fixtureModule + "/strong": {"TestSum"}}
		},
	})
	require.NoError(t, err)

	filtered := mutate.Exclude(mutants, map[string][]string{fixtureModule + "/strong": {"TestSum"}})

	results, err := mutate.Execute(t.Context(), filtered, mutate.ExecuteOptions{
		Root: c.root,
		Env:  append(os.Environ(), "GOWORK=off"),
		Jobs: 2,
	})
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, mutate.OutcomeNoCoverage, r.Outcome,
			"a mutant whose only judge was excluded must not be scored as killed")
	}
}

func TestExecuteStopsOnCancelledContext(t *testing.T) {
	c := newCorpus(t)

	mutants, err := mutate.Generate(t.Context(), mutate.GenerateOptions{
		Root:     c.root,
		Package:  fixtureModule + "/strong",
		Env:      append(os.Environ(), "GOWORK=off"),
		Eligible: c.scope.Eligible,
		Tests:    c.scope.Tests,
	})
	require.NoError(t, err)
	require.NotEmpty(t, mutants)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = mutate.Execute(ctx, mutants, mutate.ExecuteOptions{Root: c.root})

	require.ErrorIs(t, err, context.Canceled,
		"cancelling must abort rather than return results for mutants that never ran")
}
