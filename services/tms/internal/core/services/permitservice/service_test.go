package permitservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolvedRule(code string, maxWidth float64) *jurisdictionrule.Resolved {
	return &jurisdictionrule.Resolved{
		StateCode:         code,
		StateName:         code,
		VerificationState: jurisdictionrule.VerificationUnverified,
		MaxWidthFeet:      maxWidth,
		MaxHeightFeet:     13.5,
		MaxLengthFeet:     53,
		MaxWeightPounds:   80000,
		DaylightOnly:      true,
	}
}

func TestAssessJurisdictions_ReportsHeadroomForStatesTheLoadClears(t *testing.T) {
	tennessee := pulid.MustNew("us_")
	route := []permit.RouteJurisdiction{{StateID: tennessee, Sequence: 0}}
	rules := map[pulid.ID]*jurisdictionrule.Resolved{
		tennessee: resolvedRule("TN", 8.5),
	}

	assessed := assessJurisdictions(route, rules, jurisdictionrule.Measurements{
		WidthFeet:    8.25,
		HeightFeet:   13,
		LengthFeet:   48,
		WeightPounds: 74000,
	})

	require.Len(t, assessed, 1)
	assert.False(t, assessed[0].RequiresPermit)
	assert.InDelta(t, 0.25, assessed[0].WidthHeadroomFeet, 0.0001)
	assert.InDelta(t, 0.5, assessed[0].HeightHeadroomFeet, 0.0001)
	assert.InDelta(t, 5, assessed[0].LengthHeadroomFeet, 0.0001)
	assert.Equal(t, int64(6000), assessed[0].WeightHeadroomPounds)
}

func TestAssessJurisdictions_HeadroomGoesNegativeWhenOver(t *testing.T) {
	georgia := pulid.MustNew("us_")
	route := []permit.RouteJurisdiction{{StateID: georgia, Sequence: 0}}
	rules := map[pulid.ID]*jurisdictionrule.Resolved{
		georgia: resolvedRule("GA", 8.5),
	}

	assessed := assessJurisdictions(route, rules, jurisdictionrule.Measurements{
		WidthFeet:    12.5,
		HeightFeet:   13,
		LengthFeet:   48,
		WeightPounds: 90000,
	})

	require.Len(t, assessed, 1)
	assert.True(t, assessed[0].RequiresPermit)
	assert.InDelta(t, -4, assessed[0].WidthHeadroomFeet, 0.0001)
	assert.Equal(t, int64(-10000), assessed[0].WeightHeadroomPounds)
}

func TestAssessJurisdictions_PreservesTravelOrderAndCarriesProvenance(t *testing.T) {
	first, second := pulid.MustNew("us_"), pulid.MustNew("us_")
	route := []permit.RouteJurisdiction{
		{StateID: first, Sequence: 0},
		{StateID: second, Sequence: 1},
	}
	rules := map[pulid.ID]*jurisdictionrule.Resolved{
		first:  resolvedRule("GA", 8.5),
		second: resolvedRule("TN", 8.5),
	}

	assessed := assessJurisdictions(route, rules, jurisdictionrule.Measurements{WidthFeet: 8})

	require.Len(t, assessed, 2)
	assert.Equal(t, "GA", assessed[0].StateCode)
	assert.Equal(t, int16(0), assessed[0].Sequence)
	assert.Equal(t, "TN", assessed[1].StateCode)
	assert.Equal(t, int16(1), assessed[1].Sequence)
	assert.Equal(
		t,
		jurisdictionrule.VerificationUnverified,
		assessed[0].VerificationState,
	)
	assert.Equal(
		t,
		[]jurisdictionrule.RestrictionKind{jurisdictionrule.RestrictionDaylightOnly},
		assessed[0].Restrictions,
	)
}

// A state with no rule row must be omitted rather than reported with zero
// limits: a zero limit reads as "everything is oversize here", and an omitted
// state reads as "we have no data", which is the truth.
func TestAssessJurisdictions_SkipsJurisdictionsWithoutRules(t *testing.T) {
	known, unknown := pulid.MustNew("us_"), pulid.MustNew("us_")
	route := []permit.RouteJurisdiction{
		{StateID: known, Sequence: 0},
		{StateID: unknown, Sequence: 1},
	}
	rules := map[pulid.ID]*jurisdictionrule.Resolved{
		known:   resolvedRule("GA", 8.5),
		unknown: nil,
	}

	assessed := assessJurisdictions(route, rules, jurisdictionrule.Measurements{WidthFeet: 8})

	require.Len(t, assessed, 1)
	assert.Equal(t, "GA", assessed[0].StateCode)
}
