package detentionengine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	resolveCustomerID = pulid.MustNew("cus_")
	resolveLocationID = pulid.MustNew("loc_")
)

func policy(
	code string,
	mutate func(*detentionpolicy.DetentionPolicy),
) *detentionpolicy.DetentionPolicy {
	p := &detentionpolicy.DetentionPolicy{
		ID:                      pulid.MustNew("dtp_"),
		Code:                    code,
		Name:                    code,
		Status:                  detentionpolicy.PolicyStatusActive,
		ClockStartBasis:         detentionpolicy.ClockStartBasisArrival,
		LateArrivalRule:         detentionpolicy.LateArrivalRuleNoEffect,
		RoundingMode:            detentionpolicy.RoundingModeUp,
		BillingIncrementMinutes: 15,
		BillingFreeMinutes:      120,
		RateSource:              detentionpolicy.RateSourceAccessorial,
		AccessorialChargeID:     pulid.MustNew("acc_"),
		NotificationRequirement: detentionpolicy.NotificationRequirementNone,
		UnnotifiedBehavior:      detentionpolicy.UnnotifiedBehaviorBill,
		DayBoundaryMode:         detentionpolicy.CapScopePerStop,
		Currency:                "USD",
		CreatedAt:               1000,
	}
	if mutate != nil {
		mutate(p)
	}
	return p
}

func matchInput() *detentionpolicy.MatchInput {
	return &detentionpolicy.MatchInput{
		CustomerID:     resolveCustomerID,
		LocationID:     resolveLocationID,
		ShipmentTypeID: pulid.MustNew("spt_"),
		ServiceTypeID:  pulid.MustNew("svt_"),
		StopType:       shipmentvocab.StopTypeDelivery,
		ScheduleType:   shipmentvocab.StopScheduleTypeAppointment,
		Timestamp:      5000,
	}
}

func TestResolvePolicy_NoCandidates(t *testing.T) {
	t.Parallel()

	res := detentionengine.ResolvePolicy(nil, matchInput())

	assert.False(t, res.Found())
	assert.Contains(t, res.Explanation, "No detention policy")
}

func TestResolvePolicy_OrgDefaultWins_WhenNothingElseMatches(t *testing.T) {
	t.Parallel()

	orgDefault := policy("ORG", func(p *detentionpolicy.DetentionPolicy) { p.IsOrgDefault = true })
	otherCustomer := policy("OTHER", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = ptr(pulid.MustNew("cus_"))
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{otherCustomer, orgDefault},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "ORG", res.Policy.Code)
}

func TestResolvePolicy_SpecificityBeatsGeneral(t *testing.T) {
	t.Parallel()

	orgDefault := policy("ORG", func(p *detentionpolicy.DetentionPolicy) { p.IsOrgDefault = true })
	customerPolicy := policy("CUST", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})
	facilityPolicy := policy("FACILITY", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{orgDefault, customerPolicy, facilityPolicy},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "FACILITY", res.Policy.Code,
		"the customer+facility term is the most specific match")
	assert.Contains(t, res.Explanation, "FACILITY")
	assert.Contains(t, res.Explanation, "specific facility")
}

func TestResolvePolicy_PriorityOverridesSpecificity(t *testing.T) {
	t.Parallel()

	facilityPolicy := policy("FACILITY", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})
	override := policy("OVERRIDE", func(p *detentionpolicy.DetentionPolicy) {
		p.IsOrgDefault = true
		p.Priority = 100
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{facilityPolicy, override},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OVERRIDE", res.Policy.Code,
		"an explicit priority is the escape hatch above computed specificity")
}

func TestResolvePolicy_TieBreaksOnAge(t *testing.T) {
	t.Parallel()

	older := policy("OLDER", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})
	newer := policy("NEWER", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 9000
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{newer, older},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OLDER", res.Policy.Code,
		"equal priority and specificity resolve to the longest-standing term")
}

func TestResolvePolicy_IsDeterministicRegardlessOfInputOrder(t *testing.T) {
	t.Parallel()

	a := policy("A", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})
	b := policy("B", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})

	forward := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{a, b}, matchInput(),
	)
	reverse := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{b, a}, matchInput(),
	)

	require.True(t, forward.Found())
	require.True(t, reverse.Found())
	assert.Equal(t, forward.Policy.ID, reverse.Policy.ID,
		"identical candidates must resolve the same way in any order")
}

func TestResolvePolicy_EffectiveDatingExcludesExpired(t *testing.T) {
	t.Parallel()

	expired := policy("EXPIRED", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
		p.EffectiveEndDate = ptr(int64(4000))
	})
	current := policy("CURRENT", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.EffectiveStartDate = ptr(int64(1000))
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{expired, current},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "CURRENT", res.Policy.Code,
		"a more specific but expired term must not win")
}

func TestResolvePolicy_NotYetEffectiveIsExcluded(t *testing.T) {
	t.Parallel()

	future := policy("FUTURE", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.EffectiveStartDate = ptr(int64(9000))
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{future},
		matchInput(),
	)

	assert.False(t, res.Found())
	require.Len(t, res.Verdicts, 1)
	assert.Contains(t, res.Verdicts[0].Reason, "effective date range")
}

func TestResolvePolicy_VerdictsExplainEveryCandidate(t *testing.T) {
	t.Parallel()

	winner := policy("WIN", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})
	wrongCustomer := policy("WRONGCUST", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = ptr(pulid.MustNew("cus_"))
	})
	inactive := policy("INACTIVE", func(p *detentionpolicy.DetentionPolicy) {
		p.Status = detentionpolicy.PolicyStatusInactive
	})
	appointmentOnly := policy("APPT", func(p *detentionpolicy.DetentionPolicy) {
		p.AppointmentStopsOnly = true
	})

	in := matchInput()
	in.ScheduleType = shipmentvocab.StopScheduleTypeOpen

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{winner, wrongCustomer, inactive, appointmentOnly},
		in,
	)

	require.True(t, res.Found())
	assert.Equal(t, "WIN", res.Policy.Code)
	require.Len(t, res.Verdicts, 4)

	assert.True(t, res.Verdicts[0].IsWinner, "the winner sorts first")

	byCode := map[string]detentionengine.CandidateVerdict{}
	for _, v := range res.Verdicts {
		byCode[v.PolicyCode] = v
	}

	assert.False(t, byCode["WRONGCUST"].Matched)
	assert.Contains(t, byCode["WRONGCUST"].Reason, "different customer")
	assert.False(t, byCode["INACTIVE"].Matched)
	assert.Contains(t, byCode["INACTIVE"].Reason, "Inactive")
	assert.False(t, byCode["APPT"].Matched)
	assert.Contains(t, byCode["APPT"].Reason, "appointment stops")
}

func TestResolvePolicy_NilCandidatesAreSkipped(t *testing.T) {
	t.Parallel()

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{nil, policy("OK", nil), nil},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OK", res.Policy.Code)
	assert.Len(t, res.Verdicts, 1)
}

func TestResolvePolicy_ExplanationNamesTheRunnerUp(t *testing.T) {
	t.Parallel()

	facility := policy("FACILITY", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})
	customer := policy("CUSTOMER", func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{customer, facility},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Contains(t, res.Explanation, "FACILITY")
	assert.Contains(t, res.Explanation, "CUSTOMER")
	assert.Contains(t, res.Explanation, "specificity")
}

func TestResolvePolicy_SingleMatchExplanation(t *testing.T) {
	t.Parallel()

	res := detentionengine.ResolvePolicy(
		[]*detentionpolicy.DetentionPolicy{policy("ONLY", nil)},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Contains(t, res.Explanation, "only policy")
	assert.Contains(t, res.Explanation, "all freight")
}
