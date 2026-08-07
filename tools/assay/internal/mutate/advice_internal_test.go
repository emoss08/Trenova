package mutate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adviceMutant(pkg string, kind Kind, original, replacement string, plan TestPlan) Mutant {
	return Mutant{
		Package:     pkg,
		Kind:        kind,
		Original:    original,
		Replacement: replacement,
		tests:       plan,
	}
}

func adviceProfile(profiles map[string]int) TestProfile {
	return func(pkg, test string) (int, time.Duration, bool) {
		lines, ok := profiles[pkg+"."+test]

		return lines, time.Duration(lines) * time.Millisecond, ok
	}
}

// TestAdvisePrefersTheMutatedPackagesOwnTests pins the first ranking rule: a
// same-package test is the natural home for the assertion even when a
// cross-package test is more focused.
func TestAdvisePrefersTheMutatedPackagesOwnTests(t *testing.T) {
	m := adviceMutant("example.com/m/target", KindArithmetic, "+", "-", TestPlan{
		"example.com/m/target": {"TestBroad"},
		"example.com/m/user":   {"TestNarrow"},
	})

	advice := Advise(m, adviceProfile(map[string]int{
		"example.com/m/target.TestBroad": 500,
		"example.com/m/user.TestNarrow":  5,
	}))

	require.NotNil(t, advice)
	assert.Equal(t, "TestBroad", advice.Test)
	assert.Equal(t, "example.com/m/target", advice.TestPackage)
}

// TestAdvisePrefersTheMostFocusedTest pins the second rule: within the same
// package, the test covering the fewest lines has its assertions closest to
// the mutated line.
func TestAdvisePrefersTheMostFocusedTest(t *testing.T) {
	m := adviceMutant("example.com/m/target", KindBoundary, "<", "<=", TestPlan{
		"example.com/m/target": {"TestEverything", "TestJustThis"},
	})

	advice := Advise(m, adviceProfile(map[string]int{
		"example.com/m/target.TestEverything": 900,
		"example.com/m/target.TestJustThis":   12,
	}))

	require.NotNil(t, advice)
	assert.Equal(t, "TestJustThis", advice.Test)
	assert.Equal(t, 12, advice.Focus)
}

func TestAdviseRanksUnprofiledTestsLast(t *testing.T) {
	m := adviceMutant("example.com/m/target", KindEquality, "==", "!=", TestPlan{
		"example.com/m/target": {"TestKnown", "TestUnknown"},
	})

	advice := Advise(m, adviceProfile(map[string]int{
		"example.com/m/target.TestKnown": 300,
	}))

	require.NotNil(t, advice)
	assert.Equal(t, "TestKnown", advice.Test,
		"a test the index cannot profile must not outrank one it can")
}

func TestAdviseIsNilForUncoveredMutants(t *testing.T) {
	m := adviceMutant("example.com/m/target", KindBoolean, "true", "false", nil)

	assert.Nil(t, Advise(m, nil),
		"an uncovered site is a coverage gap; the coverage report owns that story")
}

// TestPrescriptionsDistinguishEveryKind pins that each mutation kind gets its
// own actionable sentence — the generic fallback wording appearing for a known
// kind would mean the table silently rotted.
func TestPrescriptionsDistinguishEveryKind(t *testing.T) {
	cases := []struct {
		kind                  Kind
		original, replacement string
		want                  string
	}{
		{KindBoundary, "<", "<=", "both sides"},
		{KindArithmetic, "+", "-", "exact result"},
		{KindEquality, "==", "!=", "both sides of this comparison"},
		{KindConnector, "&&", "||", "one operand is true and the other false"},
		{KindBoolean, "true", "false", "constant"},
		{KindBranch, "x > 0", "(x > 0) && false", "when this condition is true"},
		{KindBranch, "x > 0", "(x > 0) || true", "when this condition is false"},
		{KindIncDec, "++", "--", "final value"},
	}

	for _, tc := range cases {
		m := adviceMutant("example.com/m/p", tc.kind, tc.original, tc.replacement,
			TestPlan{"example.com/m/p": {"TestX"}})
		advice := Advise(m, nil)
		require.NotNil(t, advice)
		assert.Contains(t, advice.Prescription, tc.want, "kind %s / %s", tc.kind, tc.replacement)
	}
}

// TestGroupSurvivorsRanksTheWorstFunctionFirst pins the report order: the
// function with the most survivors is the biggest assertion gap, and a flat
// list scattered its mutants where nobody would see the cluster.
func TestGroupSurvivorsRanksTheWorstFunctionFirst(t *testing.T) {
	survivor := func(fn string, line int) Result {
		m := adviceMutant("example.com/m/p", KindArithmetic, "+", "-", nil)
		m.Function = fn
		m.File = "/src/p/file.go"
		m.Line = line

		return Result{Mutant: m, Outcome: OutcomeSurvived}
	}

	groups := GroupSurvivors([]Result{
		survivor("Lonely", 10),
		survivor("Cluster", 40),
		survivor("Cluster", 20),
		survivor("Cluster", 30),
	})

	require.Len(t, groups, 2)
	assert.Equal(t, "Cluster", groups[0].Function)
	require.Len(t, groups[0].Survivors, 3)
	assert.Equal(t, 20, groups[0].Survivors[0].Mutant.Line, "within a group, source order")
	assert.Equal(t, "Lonely", groups[1].Function)
}
