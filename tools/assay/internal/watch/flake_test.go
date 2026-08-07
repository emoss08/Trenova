package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/flake"
)

// TestCycleObservationsAttributeEveryKnownVerdict pins what watch feeds the
// flake journal: on a narrowed failing run, the named failures record as
// fails, the executed-but-unnamed tests record as passes, and unjudged or
// skipped results record nothing — an unattributable verdict is not evidence.
func TestCycleObservationsAttributeEveryKnownVerdict(t *testing.T) {
	fp := cache.Fingerprint{1, 2, 3}
	results := []RunResult{
		{
			ImportPath:  "example.com/m/red",
			Tests:       []string{"TestBroken", "TestFine"},
			Passed:      false,
			Output:      []byte("--- FAIL: TestBroken (0.00s)\n"),
			Fingerprint: fp,
		},
		{
			ImportPath:  "example.com/m/green",
			Tests:       []string{"TestOK"},
			Passed:      true,
			Fingerprint: fp,
		},
		{ImportPath: "example.com/m/whole", Passed: true, Fingerprint: fp},
		{ImportPath: "example.com/m/broken", Err: assert.AnError, Fingerprint: fp},
		{ImportPath: "example.com/m/tagged", Skipped: true, Fingerprint: fp},
	}

	observations := cycleObservations(results)

	byKey := make(map[string]flake.Verdict, len(observations))
	for _, o := range observations {
		byKey[o.Package+"."+o.Test] = o.Verdict
		assert.Equal(t, fp.String(), o.Fingerprint)
		assert.Equal(t, "watch", o.Source)
	}

	require.Len(t, observations, 3)
	assert.Equal(t, flake.VerdictFail, byKey["example.com/m/red.TestBroken"])
	assert.Equal(t, flake.VerdictPass, byKey["example.com/m/red.TestFine"],
		"a test that ran beside the failure and did not fail is a pass worth recording")
	assert.Equal(t, flake.VerdictPass, byKey["example.com/m/green.TestOK"])
}

// TestStickyRerunsProduceFlakeEvidence pins the payoff of recording watch
// verdicts at all: a failure and its sticky re-run at the same fingerprint are
// exactly a same-code conflict when the re-run passes.
func TestStickyRerunsProduceFlakeEvidence(t *testing.T) {
	fp := cache.Fingerprint{9}
	failedCycle := cycleObservations([]RunResult{{
		ImportPath:  "example.com/m/p",
		Tests:       []string{"TestJumpy"},
		Passed:      false,
		Output:      []byte("--- FAIL: TestJumpy (0.00s)\n"),
		Fingerprint: fp,
	}})
	rerunCycle := cycleObservations([]RunResult{{
		ImportPath:  "example.com/m/p",
		Tests:       []string{"TestJumpy"},
		Passed:      true,
		Fingerprint: fp,
	}})

	flakes := flake.Detect(append(failedCycle, rerunCycle...))

	require.Len(t, flakes, 1)
	assert.Equal(t, "TestJumpy", flakes[0].Test)
}

func TestQuarantinedNamesLabelsOnlyKnownFlakes(t *testing.T) {
	quarantine := flake.NewQuarantine()
	quarantine.Add(flake.QuarantineEntry{Package: "example.com/m/p", Test: "TestJumpy"})

	result := RunResult{
		ImportPath: "example.com/m/p",
		Output:     []byte("--- FAIL: TestJumpy (0.00s)\n--- FAIL: TestReal (0.00s)\n"),
	}

	assert.Equal(t, "TestJumpy", quarantinedNames(quarantine, result),
		"the honest label covers exactly the known flake, never the real failure beside it")
	assert.Empty(t, quarantinedNames(nil, result))
}
