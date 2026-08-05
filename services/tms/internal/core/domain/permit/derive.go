package permit

import (
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const SecondsPerDay = int64(86400)

// RouteJurisdiction is one leg of the route reduced to what permitting cares
// about: which jurisdiction, in what order, and how far the load travels in it.
type RouteJurisdiction struct {
	StateID       pulid.ID
	Sequence      int16
	DistanceMiles float64
}

type DeriveInput struct {
	Jurisdictions []RouteJurisdiction
	Rules         map[pulid.ID]*jurisdictionrule.Resolved
	Measurements  jurisdictionrule.Measurements
	DerivedAt     int64
}

// Derive walks the route and asks each jurisdiction what it requires. Only
// jurisdictions the load actually exceeds produce a requirement, so a legal
// load crossing eight states yields nothing rather than eight empty rows.
func Derive(input *DeriveInput) []*Requirement {
	requirements := make([]*Requirement, 0, len(input.Jurisdictions))

	for _, jurisdiction := range input.Jurisdictions {
		resolved, ok := input.Rules[jurisdiction.StateID]
		if !ok || resolved == nil {
			continue
		}

		measurements := input.Measurements
		measurements.DistanceMiles = jurisdiction.DistanceMiles

		exceedances := resolved.Evaluate(measurements)
		if len(exceedances) == 0 {
			continue
		}

		requirements = append(requirements, buildRequirement(
			jurisdiction, resolved, measurements, exceedances, input.DerivedAt,
		))
	}

	return requirements
}

func buildRequirement(
	jurisdiction RouteJurisdiction,
	resolved *jurisdictionrule.Resolved,
	measurements jurisdictionrule.Measurements,
	exceedances []jurisdictionrule.Exceedance,
	derivedAt int64,
) *Requirement {
	requirement := &Requirement{
		StateID:       jurisdiction.StateID,
		Status:        RequirementOpen,
		RouteSequence: jurisdiction.Sequence,
		Exceedances:   exceedances,
		Escorts:       resolved.Escorts(measurements),
		Restrictions:  resolved.Restrictions(),
		LeadTimeDays:  resolved.PermitLeadTimeDays,
		ValidityDays:  resolved.PermitValidityDays,
		DerivedAt:     derivedAt,
		Provenance: &RuleProvenance{
			RuleID:            resolved.RuleID,
			StateCode:         resolved.StateCode,
			StateName:         resolved.StateName,
			VerificationState: resolved.VerificationState,
			VerifiedAt:        resolved.VerifiedAt,
			SourceNote:        resolved.SourceNote,
			SourceURL:         resolved.SourceURL,
			MaxWidthFeet:      resolved.MaxWidthFeet,
			MaxHeightFeet:     resolved.MaxHeightFeet,
			MaxLengthFeet:     resolved.MaxLengthFeet,
			MaxWeightPounds:   resolved.MaxWeightPounds,
			OverriddenFields:  resolved.OverriddenFields,
			OverrideReason:    resolved.OverrideReason,
		},
	}

	if fee := resolved.EstimatedFee(measurements); !fee.IsZero() {
		requirement.EstimatedFee = decimal.NewNullDecimal(fee)
	}

	for _, exceedance := range exceedances {
		if exceedance.Superload {
			requirement.IsSuperload = true
			break
		}
	}

	return requirement
}

// MaxLeadTimeDays is the gating number for scheduling: the route can only start
// once the slowest jurisdiction on it has had time to issue.
func MaxLeadTimeDays(requirements []*Requirement) int16 {
	var maxDays int16

	for _, requirement := range requirements {
		if requirement != nil && requirement.LeadTimeDays > maxDays {
			maxDays = requirement.LeadTimeDays
		}
	}

	return maxDays
}

// EarliestFeasiblePickup answers the question that actually loses money when
// guessed: quoting a Monday pickup on a load whose permits take five days.
func EarliestFeasiblePickup(requirements []*Requirement, from int64) int64 {
	return from + int64(MaxLeadTimeDays(requirements))*SecondsPerDay
}

func TotalEstimatedFee(requirements []*Requirement) decimal.Decimal {
	total := decimal.Zero

	for _, requirement := range requirements {
		if requirement != nil && requirement.EstimatedFee.Valid {
			total = total.Add(requirement.EstimatedFee.Decimal)
		}
	}

	return total
}

// TotalEscorts counts escort vehicles across the route by taking the maximum
// per role rather than the sum. A front escort that runs Georgia through
// Nebraska is one vehicle hired for the trip, not three.
func TotalEscorts(requirements []*Requirement) int {
	perRole := make(map[jurisdictionrule.EscortRole]int, 4)

	for _, requirement := range requirements {
		if requirement == nil {
			continue
		}

		seen := make(map[jurisdictionrule.EscortRole]int, len(requirement.Escorts))
		for _, escort := range requirement.Escorts {
			seen[escort.Role]++
		}

		for role, count := range seen {
			if count > perRole[role] {
				perRole[role] = count
			}
		}
	}

	total := 0
	for _, count := range perRole {
		total += count
	}

	return total
}

func UnsatisfiedRequirements(requirements []*Requirement) []*Requirement {
	open := make([]*Requirement, 0, len(requirements))

	for _, requirement := range requirements {
		if requirement != nil && requirement.Status.BlocksDispatch() {
			open = append(open, requirement)
		}
	}

	return open
}

// MatchPermits marks each requirement satisfied by an active permit for the
// same jurisdiction that still covers the load at coversAt. Requirements
// without a covering permit are left Open so the caller can block dispatch.
func MatchPermits(
	requirements []*Requirement,
	permits []*Permit,
	coversAt int64,
) {
	byState := make(map[pulid.ID][]*Permit, len(permits))
	for _, permit := range permits {
		if permit != nil {
			byState[permit.StateID] = append(byState[permit.StateID], permit)
		}
	}

	for _, requirement := range requirements {
		if requirement == nil || requirement.Status == RequirementWaived {
			continue
		}

		requirement.Status = RequirementOpen
		requirement.SatisfiedByPermitID = nil

		candidates := byState[requirement.StateID]
		sort.Slice(candidates, func(i, j int) bool {
			return expiryOrMax(candidates[i]) > expiryOrMax(candidates[j])
		})

		for _, permit := range candidates {
			if !permit.CoversAt(coversAt) {
				continue
			}

			permitID := permit.ID
			requirement.Status = RequirementSatisfied
			requirement.SatisfiedByPermitID = &permitID
			break
		}
	}
}

func expiryOrMax(p *Permit) int64 {
	if p == nil || p.ExpiresAt == nil {
		return 0
	}
	return *p.ExpiresAt
}

// ExpiringBefore returns requirements whose satisfying permit lapses before the
// load is delivered. These are satisfied today and will not be on arrival,
// which is the failure mode a plain "has a permit" check misses entirely.
func ExpiringBefore(
	requirements []*Requirement,
	permits []*Permit,
	deliveryAt int64,
) []*Requirement {
	byID := make(map[pulid.ID]*Permit, len(permits))
	for _, permit := range permits {
		if permit != nil {
			byID[permit.ID] = permit
		}
	}

	expiring := make([]*Requirement, 0, len(requirements))

	for _, requirement := range requirements {
		if requirement == nil || requirement.SatisfiedByPermitID == nil {
			continue
		}

		permit, ok := byID[*requirement.SatisfiedByPermitID]
		if !ok {
			continue
		}

		if permit.IsExpiredAt(deliveryAt) {
			expiring = append(expiring, requirement)
		}
	}

	return expiring
}
