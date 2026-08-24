package permitservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

func waivedRequirement(stateID pulid.ID, by pulid.ID, reason string) *permit.Requirement {
	return &permit.Requirement{
		StateID:      stateID,
		Status:       permit.RequirementWaived,
		WaivedByID:   &by,
		WaiverReason: reason,
	}
}

func TestCarryWaiversForward_ReattachesAWaiverToTheReDerivedRequirement(t *testing.T) {
	georgia := pulid.MustNew("us_")
	waiver := pulid.MustNew("usr_")

	derived := []*permit.Requirement{
		{StateID: georgia, Status: permit.RequirementOpen},
	}

	carryWaiversForward(derived, []*permit.Requirement{
		waivedRequirement(georgia, waiver, "Escort booked, moving under state escort authority"),
	})

	assert.Equal(t, permit.RequirementWaived, derived[0].Status)
	require.NotNil(t, derived[0].WaivedByID)
	assert.Equal(t, waiver, *derived[0].WaivedByID)
	assert.Equal(
		t,
		"Escort booked, moving under state escort authority",
		derived[0].WaiverReason,
	)
}

func TestCarryWaiversForward_LeavesOtherJurisdictionsOpen(t *testing.T) {
	georgia, tennessee := pulid.MustNew("us_"), pulid.MustNew("us_")
	waiver := pulid.MustNew("usr_")

	derived := []*permit.Requirement{
		{StateID: georgia, Status: permit.RequirementOpen},
		{StateID: tennessee, Status: permit.RequirementOpen},
	}

	carryWaiversForward(derived, []*permit.Requirement{
		waivedRequirement(georgia, waiver, "Accepted risk, documented with the shipper"),
	})

	assert.Equal(t, permit.RequirementWaived, derived[0].Status)
	assert.Equal(t, permit.RequirementOpen, derived[1].Status)
	assert.Nil(t, derived[1].WaivedByID)
}

// A satisfied requirement must not carry forward. MatchPermits recomputes
// satisfaction from the permit rows every time, so copying the old status would
// let a permit that has since expired keep discharging the requirement.
func TestCarryWaiversForward_DoesNotCarrySatisfaction(t *testing.T) {
	georgia := pulid.MustNew("us_")
	permitID := pulid.MustNew("pmt_")

	derived := []*permit.Requirement{
		{StateID: georgia, Status: permit.RequirementOpen},
	}

	carryWaiversForward(derived, []*permit.Requirement{
		{
			StateID:             georgia,
			Status:              permit.RequirementSatisfied,
			SatisfiedByPermitID: &permitID,
		},
	})

	assert.Equal(t, permit.RequirementOpen, derived[0].Status)
	assert.Nil(t, derived[0].SatisfiedByPermitID)
}

func TestCarryWaiversForward_ToleratesNilEntries(t *testing.T) {
	georgia := pulid.MustNew("us_")
	waiver := pulid.MustNew("usr_")

	derived := []*permit.Requirement{nil, {StateID: georgia, Status: permit.RequirementOpen}}

	carryWaiversForward(derived, []*permit.Requirement{
		nil,
		waivedRequirement(georgia, waiver, "Documented exception approved by compliance"),
	})

	assert.Equal(t, permit.RequirementWaived, derived[1].Status)
}

type assessPermitRepoStub struct {
	repositories.PermitRepository

	resolveRouteStates func(*repositories.ResolveRouteStatesRequest) (map[pulid.ID]pulid.ID, error)
	listByShipment     func(*repositories.ListPermitsRequest) ([]*permit.Permit, error)
	listRequirements   func(*repositories.ListRequirementsRequest) ([]*permit.Requirement, error)
}

func (s *assessPermitRepoStub) ResolveRouteStates(
	_ context.Context, req *repositories.ResolveRouteStatesRequest,
) (map[pulid.ID]pulid.ID, error) {
	return s.resolveRouteStates(req)
}

func (s *assessPermitRepoStub) ListByShipment(
	_ context.Context, req *repositories.ListPermitsRequest,
) ([]*permit.Permit, error) {
	return s.listByShipment(req)
}

func (s *assessPermitRepoStub) ListRequirements(
	_ context.Context, req *repositories.ListRequirementsRequest,
) ([]*permit.Requirement, error) {
	return s.listRequirements(req)
}

type jurisdictionRepoStub struct {
	repositories.JurisdictionRuleRepository

	rules []*jurisdictionrule.JurisdictionRule
}

func (s *jurisdictionRepoStub) GetActiveByStateIDs(
	context.Context, *repositories.GetJurisdictionRulesRequest,
) ([]*jurisdictionrule.JurisdictionRule, error) {
	return s.rules, nil
}

func (s *jurisdictionRepoStub) GetOverrides(
	context.Context, pagination.TenantInfo,
) ([]*jurisdictionrule.Override, error) {
	return nil, nil
}

func activeRule(stateID pulid.ID) *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		ID:                pulid.MustNew("jur_"),
		StateID:           stateID,
		Status:            jurisdictionrule.StatusActive,
		VerificationState: jurisdictionrule.VerificationUnverified,
		MaxWidthFeet:      8.5,
		MaxHeightFeet:     13.5,
		MaxLengthFeet:     53,
		MaxWeightPounds:   80000,
		State:             &usstate.UsState{Abbreviation: "GA", Name: "Georgia"},
	}
}

func oversizeShipment(id, locationID pulid.ID) *shipment.Shipment {
	width := 12.5
	return &shipment.Shipment{
		ID:                id,
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		EnvelopeWidthFeet: &width,
		Moves: []*shipment.ShipmentMove{{
			Stops: []*shipment.Stop{{LocationID: locationID}},
		}},
	}
}

// The assessment is what the panel badge, the open count, and the dispatch
// advisory are built from. If it re-derives the requirement as Open after an
// operator waived it, the waiver "keeps coming back" everywhere the assessment
// feeds, no matter what the persisted rows say.
func TestAssess_ReflectsAPersistedWaiver(t *testing.T) {
	georgia := pulid.MustNew("us_")
	locationID := pulid.MustNew("loc_")
	shipmentID := pulid.MustNew("shp_")
	waiver := pulid.MustNew("usr_")
	reason := "Escort booked and route surveyed with the state"

	svc := &service{
		permitRepo: &assessPermitRepoStub{
			resolveRouteStates: func(*repositories.ResolveRouteStatesRequest) (map[pulid.ID]pulid.ID, error) {
				return map[pulid.ID]pulid.ID{locationID: georgia}, nil
			},
			listByShipment: func(*repositories.ListPermitsRequest) ([]*permit.Permit, error) {
				return nil, nil
			},
			listRequirements: func(*repositories.ListRequirementsRequest) ([]*permit.Requirement, error) {
				return []*permit.Requirement{
					waivedRequirement(georgia, waiver, reason),
				}, nil
			},
		},
		jurisdictionRepo: &jurisdictionRepoStub{rules: []*jurisdictionrule.JurisdictionRule{
			activeRule(georgia),
		}},
		l: zap.NewNop(),
	}

	assessment, err := svc.Assess(t.Context(), oversizeShipment(shipmentID, locationID))
	require.NoError(t, err)

	require.Len(t, assessment.Requirements, 1)
	assert.Equal(t, permit.RequirementWaived, assessment.Requirements[0].Status)
	assert.Equal(t, reason, assessment.Requirements[0].WaiverReason)
	require.NotNil(t, assessment.Requirements[0].WaivedByID)
	assert.Equal(t, waiver, *assessment.Requirements[0].WaivedByID)
	assert.Empty(t, assessment.Open, "a waived requirement must not read as open")
	assert.False(t, assessment.HasOpen())
}

// An unsaved shipment has no persisted rows to consult, and asking for them
// with a nil ID would be a scan of the whole tenant's requirements.
func TestAssess_SkipsPersistedLookupForUnsavedShipments(t *testing.T) {
	georgia := pulid.MustNew("us_")
	locationID := pulid.MustNew("loc_")

	svc := &service{
		permitRepo: &assessPermitRepoStub{
			resolveRouteStates: func(*repositories.ResolveRouteStatesRequest) (map[pulid.ID]pulid.ID, error) {
				return map[pulid.ID]pulid.ID{locationID: georgia}, nil
			},
			listByShipment: func(*repositories.ListPermitsRequest) ([]*permit.Permit, error) {
				return nil, nil
			},
			listRequirements: func(*repositories.ListRequirementsRequest) ([]*permit.Requirement, error) {
				panic("unexpected ListRequirements call for an unsaved shipment")
			},
		},
		jurisdictionRepo: &jurisdictionRepoStub{rules: []*jurisdictionrule.JurisdictionRule{
			activeRule(georgia),
		}},
		l: zap.NewNop(),
	}

	assessment, err := svc.Assess(t.Context(), oversizeShipment(pulid.Nil, locationID))
	require.NoError(t, err)

	require.Len(t, assessment.Requirements, 1)
	assert.Equal(t, permit.RequirementOpen, assessment.Requirements[0].Status)
}

// Superseded rows are the audit trail of prior derivations, not the current
// requirement set. Serving them to the panel shows the same jurisdiction once
// per save, which reads as the requirement coming back.
func TestListRequirements_ExcludesSupersededRows(t *testing.T) {
	var captured *repositories.ListRequirementsRequest

	svc := &service{
		permitRepo: &assessPermitRepoStub{
			listRequirements: func(req *repositories.ListRequirementsRequest) ([]*permit.Requirement, error) {
				captured = req
				return nil, nil
			},
		},
		l: zap.NewNop(),
	}

	_, err := svc.ListRequirements(t.Context(), pulid.MustNew("shp_"), pagination.TenantInfo{})
	require.NoError(t, err)

	require.NotNil(t, captured)
	assert.True(t, captured.ExcludeSuperseded)
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
