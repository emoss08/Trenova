package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These assertions guard the judgement calls in the baseline rather than the
// insert mechanics. The numbers here drive whether an oversize load is stopped
// before it leaves, so the safe direction matters more than the exact value.
func TestJurisdictionBaseline_UsesOnlyFederallyUniformLimits(t *testing.T) {
	// 23 CFR 658.15 — 102 inches on the National Network.
	assert.InDelta(t, 8.5, federalMaxWidthFeet, 0.001)
	// 23 U.S.C. 127 — Interstate gross, subject to the bridge formula.
	assert.Equal(t, 80_000, federalMaxWeightPounds)
}

// Height and length vary by state, so the baseline takes the low end. A limit
// set too high lets an illegal load dispatch with no requirement raised, which
// is the failure this engine exists to prevent; too low raises a requirement an
// operator can check and clear.
func TestJurisdictionBaseline_ErrsStrictOnStateVariableLimits(t *testing.T) {
	assert.LessOrEqual(t, conservativeMaxHeightFeet, 14.0,
		"states commonly permit 13'6\" to 14'; the baseline must not exceed the top of that range")
	assert.GreaterOrEqual(t, conservativeMaxHeightFeet, 13.5,
		"below 13'6\" every standard dry van would raise a permit requirement")
	assert.LessOrEqual(t, conservativeMaxLengthFeet, 53.0,
		"53ft is the common single-trailer maximum; above it over-length loads go unflagged")
}

// The source note is what an operator reads before trusting a number. It has to
// say what is real, what is a guess, and what is simply absent.
func TestJurisdictionBaseline_SourceNoteDisclosesItsLimits(t *testing.T) {
	for _, phrase := range []string{
		"Federal baseline only",
		"not this state's statute",
		"NOT populated",
		"mark this row Verified",
	} {
		assert.Contains(t, baselineSourceNote, phrase)
	}
	assert.NotEmpty(t, baselineSourceURL)
}

// Unverified is not a formality. A requirement derived from one of these rows
// carries the state through to the panel, which reports it to the operator.
// Seeding as Verified would assert a confirmation nobody performed.
func TestJurisdictionBaseline_SeedsEveryRowUnverified(t *testing.T) {
	seed := NewJurisdictionRulesBaselineSeed()

	require.Equal(t, "JurisdictionRulesBaseline", seed.Name())
	assert.Contains(t, seed.Environments(), common.EnvProduction)
	assert.Contains(t, seed.DependsOn(), string(seedhelpers.SeedUSStates),
		"rules key on us_states and must not run before it")

	// The constant the Run path writes.
	assert.Equal(
		t,
		jurisdictionrule.VerificationState("Unverified"),
		jurisdictionrule.VerificationUnverified,
	)
}
