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
