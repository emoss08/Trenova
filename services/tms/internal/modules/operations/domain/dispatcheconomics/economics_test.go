package dispatcheconomics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/stretchr/testify/assert"
)

func TestNormalized_KeepsAnOrganizationsChoicesAndFillsWhatItLeftOut(t *testing.T) {
	t.Parallel()

	stored := dispatcheconomics.Economics{
		EmptyMileCents: 250,
		Preferences: map[dispatcheconomics.Preference]int64{
			dispatcheconomics.PreferenceSafety: 0,
			dispatcheconomics.Preference("x"):  9,
		},
	}

	normalized := stored.Normalized()

	assert.Equal(t, int64(250), normalized.EmptyMileCents)
	assert.Zero(t, normalized.LateMinuteCents, "an explicit zero is a choice, not a gap")
	assert.Equal(t, int64(0), normalized.Preference(dispatcheconomics.PreferenceSafety))
	assert.Equal(
		t,
		new(dispatcheconomics.Default()).Preference(dispatcheconomics.PreferenceHaulFit),
		normalized.Preference(dispatcheconomics.PreferenceHaulFit),
	)
	assert.NotContains(t, normalized.Preferences, dispatcheconomics.Preference("x"))
	assert.Equal(t, 40, normalized.CandidatesPerMove)
}

func TestNormalized_GivesAnUnsetOrganizationTheDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, dispatcheconomics.Default(), new(dispatcheconomics.Economics).Normalized())
}

func TestValidate_RefusesNegativeOrOutOfRangeValues(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*dispatcheconomics.Economics){
		"negative empty mile":  func(e *dispatcheconomics.Economics) { e.EmptyMileCents = -1 },
		"huge late minute":     func(e *dispatcheconomics.Economics) { e.LateMinuteCents = 2_000_000 },
		"margin over 100":      func(e *dispatcheconomics.Economics) { e.CoverageMarginPercent = 101 },
		"too few candidates":   func(e *dispatcheconomics.Economics) { e.CandidatesPerMove = 2 },
		"too many candidates":  func(e *dispatcheconomics.Economics) { e.CandidatesPerMove = 500 },
		"negative preference":  func(e *dispatcheconomics.Economics) { e.Preferences["safety"] = -5 },
		"unknown preference":   func(e *dispatcheconomics.Economics) { e.Preferences["luck"] = 5 },
		"grace over a day":     func(e *dispatcheconomics.Economics) { e.EarlyGraceMinutes = 2_000 },
		"negative late premum": func(e *dispatcheconomics.Economics) { e.LatePremiumCents = -1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			economics := dispatcheconomics.Default()
			mutate(&economics)

			assert.Error(t, economics.Validate())
		})
	}

	assert.NoError(t, new(dispatcheconomics.Default()).Validate())
}
