package jurisdictionrule_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fptr(v float64) *float64 { return &v }

func escort(
	role jurisdictionrule.EscortRole,
	threshold float64,
) *jurisdictionrule.EscortThreshold {
	return &jurisdictionrule.EscortThreshold{
		Role:           role,
		Trigger:        jurisdictionrule.TriggerWidth,
		ThresholdValue: threshold,
	}
}

// Each state's front-escort width is the number that actually varies between
// jurisdictions, so the fixtures use the real published spread.
func stateRule(code string, frontEscortWidth float64) *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		MaxWidthFeet:       8.5,
		MaxHeightFeet:      13.5,
		MaxLengthFeet:      53,
		MaxWeightPounds:    80000,
		PermitLeadTimeDays: 2,
		PermitValidityDays: 5,
		State:              &usstate.UsState{Abbreviation: code, Name: code},
		EscortThresholds: []*jurisdictionrule.EscortThreshold{
			escort(jurisdictionrule.EscortRoleFront, frontEscortWidth),
			escort(jurisdictionrule.EscortRoleRear, 14),
			escort(jurisdictionrule.EscortRolePolice, 16),
		},
	}
}

func TestEvaluate_ReportsEveryExceedanceNotJustTheFirst(t *testing.T) {
	t.Parallel()

	resolved := stateRule("OH", 12).Resolve(nil)

	exceedances := resolved.Evaluate(jurisdictionrule.Measurements{
		WidthFeet:    12.5,
		HeightFeet:   15,
		LengthFeet:   60,
		WeightPounds: 95000,
	})

	require.Len(t, exceedances, 4)

	triggers := make([]jurisdictionrule.TriggerKind, 0, 4)
	for _, e := range exceedances {
		triggers = append(triggers, e.Trigger)
	}

	assert.ElementsMatch(t, []jurisdictionrule.TriggerKind{
		jurisdictionrule.TriggerWidth,
		jurisdictionrule.TriggerHeight,
		jurisdictionrule.TriggerLength,
		jurisdictionrule.TriggerWeight,
	}, triggers)
}

func TestEvaluate_ReportsHowFarOver(t *testing.T) {
	t.Parallel()

	resolved := stateRule("GA", 12).Resolve(nil)

	exceedances := resolved.Evaluate(jurisdictionrule.Measurements{WidthFeet: 12.5})

	require.Len(t, exceedances, 1)
	assert.InDelta(t, 8.5, exceedances[0].Limit, 0.001)
	assert.InDelta(t, 12.5, exceedances[0].Actual, 0.001)
	assert.InDelta(t, 4.0, exceedances[0].OverBy, 0.001)
	assert.Contains(t, exceedances[0].Describe("GA"), "over the GA limit")
}

func TestEvaluate_LegalLoadNeedsNoPermit(t *testing.T) {
	t.Parallel()

	resolved := stateRule("NE", 12).Resolve(nil)

	m := jurisdictionrule.Measurements{
		WidthFeet:    8.5,
		HeightFeet:   13.5,
		LengthFeet:   53,
		WeightPounds: 80000,
	}

	assert.Empty(t, resolved.Evaluate(m))
	assert.False(t, resolved.RequiresPermit(m))
}

// The published spread: a 12'6" load needs an escort in Nebraska but not in
// Montana, where the first escort does not start until 16'7".
func TestEscorts_SameWidthDiffersByJurisdiction(t *testing.T) {
	t.Parallel()

	m := jurisdictionrule.Measurements{WidthFeet: 12.5}

	nebraska := stateRule("NE", 12).Resolve(nil).Escorts(m)
	montana := stateRule("MT", 16.583).Resolve(nil).Escorts(m)

	require.Len(t, nebraska, 1)
	assert.Equal(t, jurisdictionrule.EscortRoleFront, nebraska[0].Role)
	assert.InDelta(t, 12.0, nebraska[0].Threshold, 0.001)

	assert.Empty(t, montana)
}

func TestEscorts_TexasThresholdIsHigherThanFlorida(t *testing.T) {
	t.Parallel()

	m := jurisdictionrule.Measurements{WidthFeet: 12.5}

	florida := stateRule("FL", 12.083).Resolve(nil).Escorts(m)
	texas := stateRule("TX", 14.083).Resolve(nil).Escorts(m)

	require.Len(t, florida, 1)
	assert.Empty(t, texas)
}

func TestEscorts_WiderLoadAccumulatesRoles(t *testing.T) {
	t.Parallel()

	resolved := stateRule("OH", 12).Resolve(nil)

	escorts := resolved.Escorts(jurisdictionrule.Measurements{WidthFeet: 16.5})

	require.Len(t, escorts, 3)
	assert.Equal(t, jurisdictionrule.EscortRoleFront, escorts[0].Role)
	assert.Equal(t, jurisdictionrule.EscortRolePolice, escorts[1].Role)
	assert.Equal(t, jurisdictionrule.EscortRoleRear, escorts[2].Role)
}

func TestEscorts_ThresholdIsInclusive(t *testing.T) {
	t.Parallel()

	resolved := stateRule("NE", 12).Resolve(nil)

	assert.Len(t, resolved.Escorts(jurisdictionrule.Measurements{WidthFeet: 12.0}), 1)
	assert.Empty(t, resolved.Escorts(jurisdictionrule.Measurements{WidthFeet: 11.99}))
}

func TestEscorts_DoNotDoubleCountARole(t *testing.T) {
	t.Parallel()

	rule := stateRule("CA", 12)
	rule.EscortThresholds = append(rule.EscortThresholds, &jurisdictionrule.EscortThreshold{
		Role:           jurisdictionrule.EscortRoleFront,
		Trigger:        jurisdictionrule.TriggerHeight,
		ThresholdValue: 14,
	})

	escorts := rule.Resolve(nil).Escorts(jurisdictionrule.Measurements{
		WidthFeet:  13,
		HeightFeet: 15,
	})

	fronts := 0
	for _, e := range escorts {
		if e.Role == jurisdictionrule.EscortRoleFront {
			fronts++
		}
	}

	assert.Equal(t, 1, fronts)
}

func TestEvaluate_FlagsSuperload(t *testing.T) {
	t.Parallel()

	rule := stateRule("TX", 14.083)
	rule.SuperloadWidthFeet = fptr(16)

	exceedances := rule.Resolve(nil).Evaluate(
		jurisdictionrule.Measurements{WidthFeet: 17},
	)

	require.Len(t, exceedances, 1)
	assert.True(t, exceedances[0].Superload)
}

func TestResolve_OverrideTightensAndRecordsProvenance(t *testing.T) {
	t.Parallel()

	rule := stateRule("MT", 16.583)
	override := &jurisdictionrule.Override{
		MaxWidthFeet: fptr(8.0),
		DaylightOnly: func() *bool { b := true; return &b }(),
		Reason:       "Our insurer requires permits above 8 feet regardless of state law.",
	}

	resolved := rule.Resolve(override)

	assert.InDelta(t, 8.0, resolved.MaxWidthFeet, 0.001)
	assert.True(t, resolved.DaylightOnly)
	assert.ElementsMatch(t,
		[]string{"maxWidthFeet", "daylightOnly"}, resolved.OverriddenFields)
	assert.Contains(t, resolved.OverrideReason, "insurer")

	assert.True(t, resolved.RequiresPermit(jurisdictionrule.Measurements{WidthFeet: 8.4}))
}

func TestResolve_UnusedOverrideLeavesNoProvenance(t *testing.T) {
	t.Parallel()

	resolved := stateRule("IL", 12).Resolve(&jurisdictionrule.Override{
		Reason: "Placeholder with no actual changes recorded yet.",
	})

	assert.Nil(t, resolved.OverriddenFields)
	assert.Empty(t, resolved.OverrideReason)
	assert.InDelta(t, 8.5, resolved.MaxWidthFeet, 0.001)
}

func TestEstimatedFee_CombinesBaseAndPerMile(t *testing.T) {
	t.Parallel()

	rule := stateRule("TX", 14.083)
	rule.PermitBaseFee = decimal.NewNullDecimal(decimal.NewFromInt(60))
	rule.PermitPerMileFee = decimal.NewNullDecimal(decimal.NewFromFloat(0.25))

	fee := rule.Resolve(nil).EstimatedFee(
		jurisdictionrule.Measurements{DistanceMiles: 400},
	)

	assert.True(t, fee.Equal(decimal.NewFromInt(160)), "expected 160, got %s", fee)
}

func TestEstimatedFee_ZeroWithoutConfiguredFees(t *testing.T) {
	t.Parallel()

	fee := stateRule("WY", 12).Resolve(nil).EstimatedFee(
		jurisdictionrule.Measurements{DistanceMiles: 400},
	)

	assert.True(t, fee.IsZero())
}

func TestRestrictions_ReflectsResolvedFlags(t *testing.T) {
	t.Parallel()

	rule := stateRule("OH", 12)
	rule.DaylightOnly = true
	rule.RushHourRestricted = true
	rule.HolidayRestricted = true

	assert.ElementsMatch(t, []jurisdictionrule.RestrictionKind{
		jurisdictionrule.RestrictionDaylightOnly,
		jurisdictionrule.RestrictionRushHour,
		jurisdictionrule.RestrictionHoliday,
	}, rule.Resolve(nil).Restrictions())
}

func TestIsEffectiveAt_RespectsWindow(t *testing.T) {
	t.Parallel()

	start := int64(100)
	end := int64(200)

	rule := stateRule("IA", 12)
	rule.EffectiveStartDate = &start
	rule.EffectiveEndDate = &end

	assert.False(t, rule.IsEffectiveAt(50))
	assert.True(t, rule.IsEffectiveAt(150))
	assert.False(t, rule.IsEffectiveAt(250))
}

func TestSeededDataIsUnverifiedByDefault(t *testing.T) {
	t.Parallel()

	rule := new(jurisdictionrule.JurisdictionRule)
	require.NoError(t, rule.BeforeAppendModel(t.Context(), nil))

	assert.Equal(t, jurisdictionrule.VerificationUnverified, rule.VerificationState)
	assert.False(t, rule.VerificationState.IsTrusted())
}

func iptr(v int64) *int64   { return &v }
func i16ptr(v int16) *int16 { return &v }
func bptr(v bool) *bool     { return &v }

// An override exists so a carrier can run a tighter posture than the statute
// without editing shared reference data. It must not be able to run a looser
// one: a permissive override would tell the engine an illegal load is legal, so
// no requirement is derived, no hold is raised, and the load leaves.
//
// These assertions are written from that contract, not from what Resolve
// currently does.
func TestResolve_AnOverrideCannotRaiseALimit(t *testing.T) {
	rule := stateRule("GA", 12)

	resolved := rule.Resolve(&jurisdictionrule.Override{
		Reason:          "Attempt to permit a wider load than the state allows",
		MaxWidthFeet:    fptr(20),
		MaxHeightFeet:   fptr(18),
		MaxLengthFeet:   fptr(90),
		MaxWeightPounds: iptr(120000),
	})

	assert.InDelta(t, 8.5, resolved.MaxWidthFeet, 0.001)
	assert.InDelta(t, 13.5, resolved.MaxHeightFeet, 0.001)
	assert.InDelta(t, 53, resolved.MaxLengthFeet, 0.001)
	assert.Equal(t, int64(80000), resolved.MaxWeightPounds)
}

func TestResolve_AnOverrideCanLowerALimit(t *testing.T) {
	rule := stateRule("GA", 12)

	resolved := rule.Resolve(&jurisdictionrule.Override{
		Reason:          "Our trailers are narrower than the state maximum",
		MaxWidthFeet:    fptr(8),
		MaxWeightPounds: iptr(70000),
	})

	assert.InDelta(t, 8, resolved.MaxWidthFeet, 0.001)
	assert.Equal(t, int64(70000), resolved.MaxWeightPounds)
	assert.Contains(t, resolved.OverriddenFields, "maxWidthFeet")
	assert.Contains(t, resolved.OverriddenFields, "maxWeightPounds")
}

// A limit the override tried and failed to raise is not an applied override.
// Listing it would tell an operator their stricter posture is in force when the
// statutory number is what the engine used.
func TestResolve_ARejectedLoosenIsNotReportedAsApplied(t *testing.T) {
	rule := stateRule("GA", 12)

	resolved := rule.Resolve(&jurisdictionrule.Override{
		Reason:        "Mixed: one tighter, one looser",
		MaxWidthFeet:  fptr(20),
		MaxHeightFeet: fptr(12),
	})

	assert.NotContains(t, resolved.OverriddenFields, "maxWidthFeet")
	assert.Contains(t, resolved.OverriddenFields, "maxHeightFeet")
}

// An override equal to the statutory limit changes nothing. Reporting it as an
// applied override would tell an operator a stricter posture is in force when
// the engine is using the statutory number either way.
func TestResolve_AnOverrideEqualToTheLimitIsNotApplied(t *testing.T) {
	rule := stateRule("GA", 12)

	resolved := rule.Resolve(&jurisdictionrule.Override{
		Reason:       "Restates the statutory width without changing it",
		MaxWidthFeet: fptr(8.5),
	})

	assert.InDelta(t, 8.5, resolved.MaxWidthFeet, 0.001)
	assert.NotContains(t, resolved.OverriddenFields, "maxWidthFeet")
	// Nothing was applied, so the reason must not surface either.
	assert.Empty(t, resolved.OverrideReason)
}

// Lead time runs the other way: needing more notice is the conservative
// direction, so an override may lengthen it but not shorten it.
func TestResolve_LeadTimeOnlyLengthens(t *testing.T) {
	rule := stateRule("GA", 12)

	longer := rule.Resolve(&jurisdictionrule.Override{
		Reason:             "Our permit desk needs a week",
		PermitLeadTimeDays: i16ptr(7),
	})
	assert.Equal(t, int16(7), longer.PermitLeadTimeDays)

	shorter := rule.Resolve(&jurisdictionrule.Override{
		Reason:             "Attempt to quote a sooner pickup than the state can issue",
		PermitLeadTimeDays: i16ptr(0),
	})
	assert.Equal(t, int16(2), shorter.PermitLeadTimeDays)
}

// A restriction the state imposes cannot be cleared by an override; a carrier
// may only add one the state does not impose.
func TestResolve_RestrictionsCanBeAddedButNotCleared(t *testing.T) {
	rule := stateRule("GA", 12)
	rule.DaylightOnly = true
	rule.HolidayRestricted = true

	cleared := rule.Resolve(&jurisdictionrule.Override{
		Reason:            "Attempt to move at night against the state restriction",
		DaylightOnly:      bptr(false),
		HolidayRestricted: bptr(false),
	})
	assert.True(t, cleared.DaylightOnly)
	assert.True(t, cleared.HolidayRestricted)

	unrestricted := stateRule("TN", 12)
	added := unrestricted.Resolve(&jurisdictionrule.Override{
		Reason:       "We do not run oversize at night regardless of the state",
		DaylightOnly: bptr(true),
	})
	assert.True(t, added.DaylightOnly)
	assert.Contains(t, added.OverriddenFields, "daylightOnly")
}
