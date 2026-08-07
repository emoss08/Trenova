package mutate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/mutate"
)

// TestAdviseNamesTheCoveringTest pins the end-to-end claim: every survivor's
// advice names a real covering test with a real profile from the index, and a
// prescription — the difference between a metric and a to-do item.
func TestAdviseNamesTheCoveringTest(t *testing.T) {
	c := newCorpus(t)
	results, err := mutate.Execute(t.Context(), c.generate(t, "weak"), c.executeOptions())
	require.NoError(t, err)
	requireNoBuildFailures(t, results)

	survivors := 0
	for i := range results {
		if results[i].Outcome != mutate.OutcomeSurvived {
			continue
		}
		survivors++
		advice := mutate.Advise(results[i].Mutant, c.scope.TestProfile)

		require.NotNil(t, advice, "a covered survivor must carry advice")
		assert.Equal(t, "TestDiscountIsNotNegative", advice.Test,
			"the only test covering Discount is the place to add the assertion")
		assert.Equal(t, fixtureModule+"/weak", advice.TestPackage)
		assert.Positive(t, advice.Focus, "the index knows how much that test covers")
		assert.NotEmpty(t, advice.Prescription)
	}
	require.Positive(t, survivors, "the weak corpus exists to produce survivors")
}
