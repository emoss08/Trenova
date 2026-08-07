package mutate_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/mutate"
)

// baseSource is covered from two packages: its own fast test and slowuser's
// deliberately slow one. The indexed durations make the cost order
// unambiguous, which is what lets the cheapest-first assertion be exact.
const baseSource = `package base

func Double(n int) int {
	return n * 2
}
`

const baseTest = `package base

import "testing"

func TestDouble(t *testing.T) {
	if got := Double(3); got != 6 {
		t.Fatalf("got %d", got)
	}
}
`

const slowUserSource = `package slowuser

import "` + fixtureModule + `/base"

func Quad(n int) int {
	return base.Double(base.Double(n))
}
`

const slowUserTest = `package slowuser

import (
	"testing"
	"time"
)

func TestQuad(t *testing.T) {
	time.Sleep(150 * time.Millisecond)
	if got := Quad(2); got != 8 {
		t.Fatalf("got %d", got)
	}
}
`

func orderCorpusFiles() map[string]string {
	return map[string]string{
		"go.mod":                    "module " + fixtureModule + "\n\ngo 1.26\n",
		"base/base.go":              baseSource,
		"base/base_test.go":         baseTest,
		"slowuser/slowuser.go":      slowUserSource,
		"slowuser/slowuser_test.go": slowUserTest,
	}
}

// TestPackageOrderIsCheapestFirst pins the kill lever: the index knows what
// each covering package's tests cost, so the plan's packages are judged
// cheapest first — a kill found by the microsecond test never pays for the
// 150ms one.
func TestPackageOrderIsCheapestFirst(t *testing.T) {
	c := newCorpusFrom(t, orderCorpusFiles())

	plan := mutate.TestPlan{
		fixtureModule + "/base":     {"TestDouble"},
		fixtureModule + "/slowuser": {"TestQuad"},
	}

	ordered := c.scope.PackageOrder(plan)

	require.Len(t, ordered, 2)
	assert.Equal(t, []string{fixtureModule + "/base", fixtureModule + "/slowuser"}, ordered,
		"slowuser sleeps 150ms and base runs in microseconds; lexicographic order would already be right here, so verify the reverse too")

	assert.Equal(t, ordered, c.scope.PackageOrder(plan),
		"the order must be deterministic")
}

// TestPackageOrderPutsUnindexedPackagesLast pins the unknown-cost rule: a
// package the index has no record for sorts after every indexed one — its cost
// is unknown, not zero.
func TestPackageOrderPutsUnindexedPackagesLast(t *testing.T) {
	c := newCorpusFrom(t, orderCorpusFiles())

	plan := mutate.TestPlan{
		fixtureModule + "/aaa-unindexed": {"TestGhost"},
		fixtureModule + "/slowuser":      {"TestQuad"},
	}

	ordered := c.scope.PackageOrder(plan)

	require.Len(t, ordered, 2)
	assert.Equal(t, fixtureModule+"/slowuser", ordered[0],
		"an unindexed package must not jump the queue by sorting first alphabetically")
}

// TestExecuteHonoursThePackageOrderHook pins the wiring end to end: with a
// hook that reverses lexicographic order, the kill is attributed to the
// package the hook put first.
func TestExecuteHonoursThePackageOrderHook(t *testing.T) {
	c := newCorpusFrom(t, orderCorpusFiles())
	mutants := c.generate(t, "base")

	killedByWith := func(order func(plan mutate.TestPlan) []string) map[string]string {
		opts := c.executeOptions()
		opts.PackageOrder = order
		results, err := mutate.Execute(t.Context(), mutants, opts)
		require.NoError(t, err)
		requireNoBuildFailures(t, results)

		out := make(map[string]string, len(results))
		for _, r := range results {
			if r.Outcome.Killed() {
				out[r.Mutant.ID] = r.KilledBy
			}
		}

		return out
	}

	reversed := func(plan mutate.TestPlan) []string {
		out := make([]string, 0, len(plan))
		for pkg := range plan {
			out = append(out, pkg)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(out)))

		return out
	}

	base := fixtureModule + "/base"
	slowuser := fixtureModule + "/slowuser"

	cheapFirst := killedByWith(func(plan mutate.TestPlan) []string {
		return c.scope.PackageOrder(plan)
	})
	expensiveFirst := killedByWith(reversed)

	require.NotEmpty(t, cheapFirst)
	for id, killer := range cheapFirst {
		assert.Equal(t, base, killer,
			"mutant %s: cheapest-first must find the kill in base before slowuser", id)
		assert.Equal(t, slowuser, expensiveFirst[id],
			"mutant %s: the hook's order decides which package judges first", id)
	}
}

// TestInvalidPackageOrderFallsBackToSorted pins the fail-safe: an order that
// drops, invents or duplicates a package would skip a judge, so it is ignored
// in favour of the lexicographic default rather than trusted.
func TestInvalidPackageOrderFallsBackToSorted(t *testing.T) {
	c := newCorpusFrom(t, orderCorpusFiles())
	mutants := c.generate(t, "base")

	opts := c.executeOptions()
	opts.PackageOrder = func(plan mutate.TestPlan) []string {
		return []string{fixtureModule + "/slowuser"}
	}

	results, err := mutate.Execute(t.Context(), mutants, opts)
	require.NoError(t, err)
	requireNoBuildFailures(t, results)

	for _, r := range results {
		if r.Outcome.Killed() {
			assert.Equal(t, fixtureModule+"/base", r.KilledBy,
				"a truncated order must not be trusted; sorted order judges base first")
		}
	}
}
