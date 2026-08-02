package permit_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jurisdiction struct {
	id       pulid.ID
	resolved *jurisdictionrule.Resolved
}

func newJurisdiction(code string, frontEscortWidth float64, leadDays int16) jurisdiction {
	id := pulid.MustNew("ust_")

	return jurisdiction{
		id: id,
		resolved: &jurisdictionrule.Resolved{
			RuleID:             pulid.MustNew("jr_"),
			StateID:            id,
			StateCode:          code,
			StateName:          code,
			VerificationState:  jurisdictionrule.VerificationUnverified,
			MaxWidthFeet:       8.5,
			MaxHeightFeet:      13.5,
			MaxLengthFeet:      53,
			MaxWeightPounds:    80000,
			PermitLeadTimeDays: leadDays,
			PermitValidityDays: 5,
			EscortThresholds: []*jurisdictionrule.EscortThreshold{
				{
					Role:           jurisdictionrule.EscortRoleFront,
					Trigger:        jurisdictionrule.TriggerWidth,
					ThresholdValue: frontEscortWidth,
				},
			},
		},
	}
}

func deriveAcross(
	js []jurisdiction,
	m jurisdictionrule.Measurements,
) []*permit.Requirement {
	rules := make(map[pulid.ID]*jurisdictionrule.Resolved, len(js))
	route := make([]permit.RouteJurisdiction, 0, len(js))

	for idx, j := range js {
		rules[j.id] = j.resolved
		route = append(route, permit.RouteJurisdiction{
			StateID:       j.id,
			Sequence:      int16(idx),
			DistanceMiles: 100,
		})
	}

	return permit.Derive(permit.DeriveInput{
		Jurisdictions: route,
		Rules:         rules,
		Measurements:  m,
		DerivedAt:     1000,
	})
}

// The golden case from the plan: a 12'6" wide, 15' overall, 95,000 lb load
// routed Georgia to Ohio to Nebraska.
func TestDerive_GoldenMultiStateRoute(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{
		newJurisdiction("GA", 12, 2),
		newJurisdiction("OH", 12, 2),
		newJurisdiction("NE", 12, 3),
	}

	requirements := deriveAcross(route, jurisdictionrule.Measurements{
		WidthFeet:    12.5,
		HeightFeet:   15,
		LengthFeet:   53,
		WeightPounds: 95000,
	})

	require.Len(t, requirements, 3)

	for _, requirement := range requirements {
		assert.Equal(t, permit.RequirementOpen, requirement.Status)
		assert.True(t, requirement.RequiresEscorts())
		assert.Len(t, requirement.Exceedances, 3)
	}

	assert.Equal(t, "GA", requirements[0].StateCode())
	assert.Equal(t, "NE", requirements[2].StateCode())
	assert.Equal(t, int16(3), permit.MaxLeadTimeDays(requirements))
}

func TestDerive_LegalLoadProducesNoRequirements(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{
			newJurisdiction("GA", 12, 2),
			newJurisdiction("OH", 12, 2),
			newJurisdiction("NE", 12, 3),
		},
		jurisdictionrule.Measurements{
			WidthFeet:    8.5,
			HeightFeet:   13.5,
			LengthFeet:   53,
			WeightPounds: 80000,
		},
	)

	assert.Empty(t, requirements)
}

// The published escort spread: the same width needs an escort in Nebraska and
// none in Montana.
func TestDerive_EscortsDifferByJurisdictionAtTheSameWidth(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{
			newJurisdiction("NE", 12, 2),
			newJurisdiction("MT", 16.583, 2),
		},
		jurisdictionrule.Measurements{WidthFeet: 12.5},
	)

	require.Len(t, requirements, 2)
	assert.True(t, requirements[0].RequiresEscorts(), "Nebraska should require an escort")
	assert.False(t, requirements[1].RequiresEscorts(), "Montana should not at 12.5 ft")
}

func TestDerive_SkipsJurisdictionsWithoutRules(t *testing.T) {
	t.Parallel()

	known := newJurisdiction("GA", 12, 2)

	requirements := permit.Derive(permit.DeriveInput{
		Jurisdictions: []permit.RouteJurisdiction{
			{StateID: known.id, Sequence: 0},
			{StateID: pulid.MustNew("ust_"), Sequence: 1},
		},
		Rules:        map[pulid.ID]*jurisdictionrule.Resolved{known.id: known.resolved},
		Measurements: jurisdictionrule.Measurements{WidthFeet: 12.5},
		DerivedAt:    1000,
	})

	require.Len(t, requirements, 1)
	assert.Equal(t, "GA", requirements[0].StateCode())
}

func TestDerive_CarriesUnverifiedProvenance(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{newJurisdiction("GA", 12, 2)},
		jurisdictionrule.Measurements{WidthFeet: 12.5},
	)

	require.Len(t, requirements, 1)
	require.NotNil(t, requirements[0].Provenance)
	assert.False(t, requirements[0].Provenance.IsTrusted())
	assert.Equal(t,
		jurisdictionrule.VerificationUnverified,
		requirements[0].Provenance.VerificationState)
}

func TestDerive_DescribeNamesTheTriggeringDimension(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{newJurisdiction("OH", 12, 2)},
		jurisdictionrule.Measurements{WidthFeet: 12.5},
	)

	require.Len(t, requirements, 1)
	assert.Contains(t, requirements[0].Describe(), "OH permit required")
	assert.Contains(t, requirements[0].Describe(), "width 4.00ft over")
}

func TestEarliestFeasiblePickup_HonorsSlowestJurisdiction(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{
			newJurisdiction("TX", 14.083, 1),
			newJurisdiction("UT", 12, 5),
		},
		jurisdictionrule.Measurements{WidthFeet: 12.5},
	)

	earliest := permit.EarliestFeasiblePickup(requirements, 1_000_000)

	assert.Equal(t, int64(1_000_000+5*permit.SecondsPerDay), earliest)
}

func TestEarliestFeasiblePickup_NoRequirementsMeansNoDelay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(500), permit.EarliestFeasiblePickup(nil, 500))
}

// Escorts are hired per trip, not per state: a front escort running three
// states is one vehicle, not three.
func TestTotalEscorts_TakesMaximumPerRoleNotSum(t *testing.T) {
	t.Parallel()

	requirements := deriveAcross(
		[]jurisdiction{
			newJurisdiction("GA", 12, 2),
			newJurisdiction("OH", 12, 2),
			newJurisdiction("NE", 12, 2),
		},
		jurisdictionrule.Measurements{WidthFeet: 12.5},
	)

	require.Len(t, requirements, 3)
	assert.Equal(t, 1, permit.TotalEscorts(requirements))
}

func TestTotalEstimatedFee_SumsAcrossJurisdictions(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{
		newJurisdiction("GA", 12, 2),
		newJurisdiction("OH", 12, 2),
	}
	route[0].resolved.PermitBaseFee = decimal.NewNullDecimal(decimal.NewFromInt(30))
	route[1].resolved.PermitBaseFee = decimal.NewNullDecimal(decimal.NewFromInt(25))

	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})

	total := permit.TotalEstimatedFee(requirements)
	assert.True(t, total.Equal(decimal.NewFromInt(55)), "expected 55, got %s", total)
}

func activePermit(stateID pulid.ID, expires int64) *permit.Permit {
	return &permit.Permit{
		ID:        pulid.MustNew("pmt_"),
		StateID:   stateID,
		Status:    permit.StatusActive,
		ExpiresAt: &expires,
	}
}

func TestMatchPermits_SatisfiesCoveredJurisdictions(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2), newJurisdiction("OH", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})
	require.Len(t, requirements, 2)

	permit.MatchPermits(requirements, []*permit.Permit{
		activePermit(route[0].id, 5000),
	}, 2000)

	assert.Equal(t, permit.RequirementSatisfied, requirements[0].Status)
	require.NotNil(t, requirements[0].SatisfiedByPermitID)
	assert.Equal(t, permit.RequirementOpen, requirements[1].Status)

	assert.Len(t, permit.UnsatisfiedRequirements(requirements), 1)
}

// A pending application is not authorization to move.
func TestMatchPermits_PendingPermitDoesNotSatisfy(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})

	pending := activePermit(route[0].id, 5000)
	pending.Status = permit.StatusPending

	permit.MatchPermits(requirements, []*permit.Permit{pending}, 2000)

	assert.Equal(t, permit.RequirementOpen, requirements[0].Status)
}

func TestMatchPermits_PermitExpiredAtCoverageTimeDoesNotSatisfy(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})

	permit.MatchPermits(requirements, []*permit.Permit{
		activePermit(route[0].id, 1500),
	}, 2000)

	assert.Equal(t, permit.RequirementOpen, requirements[0].Status)
}

func TestMatchPermits_PrefersTheLongestLivedPermit(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})

	shortLived := activePermit(route[0].id, 2500)
	longLived := activePermit(route[0].id, 9000)

	permit.MatchPermits(requirements, []*permit.Permit{shortLived, longLived}, 2000)

	require.NotNil(t, requirements[0].SatisfiedByPermitID)
	assert.Equal(t, longLived.ID, *requirements[0].SatisfiedByPermitID)
}

func TestMatchPermits_LeavesWaivedRequirementsAlone(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})
	requirements[0].Status = permit.RequirementWaived

	permit.MatchPermits(requirements, nil, 2000)

	assert.Equal(t, permit.RequirementWaived, requirements[0].Status)
	assert.Empty(t, permit.UnsatisfiedRequirements(requirements))
}

// The failure a plain "has a permit" check misses: covered at dispatch, lapsed
// on arrival.
func TestExpiringBefore_CatchesPermitsThatLapseMidTrip(t *testing.T) {
	t.Parallel()

	route := []jurisdiction{newJurisdiction("GA", 12, 2)}
	requirements := deriveAcross(route, jurisdictionrule.Measurements{WidthFeet: 12.5})

	issued := activePermit(route[0].id, 3000)
	permit.MatchPermits(requirements, []*permit.Permit{issued}, 2000)
	require.Equal(t, permit.RequirementSatisfied, requirements[0].Status)

	expiring := permit.ExpiringBefore(requirements, []*permit.Permit{issued}, 4000)

	require.Len(t, expiring, 1)
	assert.Equal(t, "GA", expiring[0].StateCode())

	assert.Empty(t, permit.ExpiringBefore(requirements, []*permit.Permit{issued}, 2900))
}

func TestPermitCoversAt_RespectsIssueAndExpiry(t *testing.T) {
	t.Parallel()

	issued := int64(1000)
	expires := int64(2000)

	p := &permit.Permit{Status: permit.StatusActive, IssuedAt: &issued, ExpiresAt: &expires}

	assert.False(t, p.CoversAt(999))
	assert.True(t, p.CoversAt(1500))
	assert.False(t, p.CoversAt(2001))
}
