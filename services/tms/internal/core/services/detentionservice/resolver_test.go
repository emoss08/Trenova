package detentionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	resolveCustomerID = pulid.MustNew("cus_")
	resolveLocationID = pulid.MustNew("loc_")
)

func policy(code string, mutate func(*detention.DetentionPolicy)) *detention.DetentionPolicy {
	p := &detention.DetentionPolicy{
		ID:                      pulid.MustNew("dtp_"),
		Code:                    code,
		Name:                    code,
		Status:                  detention.PolicyStatusActive,
		ClockStartBasis:         detention.ClockStartBasisArrival,
		LateArrivalRule:         detention.LateArrivalRuleNoEffect,
		RoundingMode:            detention.RoundingModeUp,
		BillingIncrementMinutes: 15,
		BillingFreeMinutes:      120,
		RateSource:              detention.RateSourceAccessorial,
		AccessorialChargeID:     pulid.MustNew("acc_"),
		NotificationRequirement: detention.NotificationRequirementNone,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
		DayBoundaryMode:         detention.CapScopePerStop,
		Currency:                "USD",
		CreatedAt:               1000,
	}
	if mutate != nil {
		mutate(p)
	}
	return p
}

func matchInput() detention.MatchInput {
	return detention.MatchInput{
		CustomerID:     resolveCustomerID,
		LocationID:     resolveLocationID,
		ShipmentTypeID: pulid.MustNew("spt_"),
		ServiceTypeID:  pulid.MustNew("svt_"),
		StopType:       shipment.StopTypeDelivery,
		ScheduleType:   shipment.StopScheduleTypeAppointment,
		Timestamp:      5000,
	}
}

func TestResolvePolicy_NoCandidates(t *testing.T) {
	t.Parallel()

	res := detentionservice.ResolvePolicy(nil, matchInput())

	assert.False(t, res.Found())
	assert.Contains(t, res.Explanation, "No detention policy")
}

func TestResolvePolicy_OrgDefaultWins_WhenNothingElseMatches(t *testing.T) {
	t.Parallel()

	orgDefault := policy("ORG", func(p *detention.DetentionPolicy) { p.IsOrgDefault = true })
	otherCustomer := policy("OTHER", func(p *detention.DetentionPolicy) {
		p.CustomerID = ptr(pulid.MustNew("cus_"))
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{otherCustomer, orgDefault},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "ORG", res.Policy.Code)
}

func TestResolvePolicy_SpecificityBeatsGeneral(t *testing.T) {
	t.Parallel()

	orgDefault := policy("ORG", func(p *detention.DetentionPolicy) { p.IsOrgDefault = true })
	customerPolicy := policy("CUST", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})
	facilityPolicy := policy("FACILITY", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{orgDefault, customerPolicy, facilityPolicy},
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

	facilityPolicy := policy("FACILITY", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})
	override := policy("OVERRIDE", func(p *detention.DetentionPolicy) {
		p.IsOrgDefault = true
		p.Priority = 100
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{facilityPolicy, override},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OVERRIDE", res.Policy.Code,
		"an explicit priority is the escape hatch above computed specificity")
}

func TestResolvePolicy_TieBreaksOnAge(t *testing.T) {
	t.Parallel()

	older := policy("OLDER", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})
	newer := policy("NEWER", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 9000
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{newer, older},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OLDER", res.Policy.Code,
		"equal priority and specificity resolve to the longest-standing term")
}

func TestResolvePolicy_IsDeterministicRegardlessOfInputOrder(t *testing.T) {
	t.Parallel()

	a := policy("A", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})
	b := policy("B", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.CreatedAt = 1000
	})

	forward := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{a, b}, matchInput(),
	)
	reverse := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{b, a}, matchInput(),
	)

	require.True(t, forward.Found())
	require.True(t, reverse.Found())
	assert.Equal(t, forward.Policy.ID, reverse.Policy.ID,
		"identical candidates must resolve the same way in any order")
}

func TestResolvePolicy_EffectiveDatingExcludesExpired(t *testing.T) {
	t.Parallel()

	expired := policy("EXPIRED", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
		p.EffectiveEndDate = ptr(int64(4000))
	})
	current := policy("CURRENT", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.EffectiveStartDate = ptr(int64(1000))
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{expired, current},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "CURRENT", res.Policy.Code,
		"a more specific but expired term must not win")
}

func TestResolvePolicy_NotYetEffectiveIsExcluded(t *testing.T) {
	t.Parallel()

	future := policy("FUTURE", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.EffectiveStartDate = ptr(int64(9000))
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{future},
		matchInput(),
	)

	assert.False(t, res.Found())
	require.Len(t, res.Verdicts, 1)
	assert.Contains(t, res.Verdicts[0].Reason, "effective date range")
}

func TestResolvePolicy_VerdictsExplainEveryCandidate(t *testing.T) {
	t.Parallel()

	winner := policy("WIN", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})
	wrongCustomer := policy("WRONGCUST", func(p *detention.DetentionPolicy) {
		p.CustomerID = ptr(pulid.MustNew("cus_"))
	})
	inactive := policy("INACTIVE", func(p *detention.DetentionPolicy) {
		p.Status = detention.PolicyStatusInactive
	})
	appointmentOnly := policy("APPT", func(p *detention.DetentionPolicy) {
		p.AppointmentStopsOnly = true
	})

	in := matchInput()
	in.ScheduleType = shipment.StopScheduleTypeOpen

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{winner, wrongCustomer, inactive, appointmentOnly},
		in,
	)

	require.True(t, res.Found())
	assert.Equal(t, "WIN", res.Policy.Code)
	require.Len(t, res.Verdicts, 4)

	assert.True(t, res.Verdicts[0].IsWinner, "the winner sorts first")

	byCode := map[string]detentionservice.CandidateVerdict{}
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

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{nil, policy("OK", nil), nil},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Equal(t, "OK", res.Policy.Code)
	assert.Len(t, res.Verdicts, 1)
}

func TestResolvePolicy_ExplanationNamesTheRunnerUp(t *testing.T) {
	t.Parallel()

	facility := policy("FACILITY", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
		p.LocationID = &resolveLocationID
	})
	customer := policy("CUSTOMER", func(p *detention.DetentionPolicy) {
		p.CustomerID = &resolveCustomerID
	})

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{customer, facility},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Contains(t, res.Explanation, "FACILITY")
	assert.Contains(t, res.Explanation, "CUSTOMER")
	assert.Contains(t, res.Explanation, "specificity")
}

func TestResolvePolicy_SingleMatchExplanation(t *testing.T) {
	t.Parallel()

	res := detentionservice.ResolvePolicy(
		[]*detention.DetentionPolicy{policy("ONLY", nil)},
		matchInput(),
	)

	require.True(t, res.Found())
	assert.Contains(t, res.Explanation, "only policy")
	assert.Contains(t, res.Explanation, "all freight")
}
