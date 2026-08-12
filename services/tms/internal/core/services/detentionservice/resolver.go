package detentionservice

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// CandidateVerdict records why a policy did or did not govern a stop. Surfacing
// the losers is what turns "why is it charging the wrong rate" from a support
// ticket into a self-serve answer.
type CandidateVerdict struct {
	PolicyID         pulid.ID `json:"policyId"`
	PolicyCode       string   `json:"policyCode"`
	PolicyName       string   `json:"policyName"`
	Matched          bool     `json:"matched"`
	Priority         int16    `json:"priority"`
	SpecificityScore int32    `json:"specificityScore"`
	Reason           string   `json:"reason"`
	IsWinner         bool     `json:"isWinner"`
}

// Resolution is the outcome of matching a stop against every candidate policy.
type Resolution struct {
	Policy      *detention.DetentionPolicy `json:"-"`
	Verdicts    []CandidateVerdict         `json:"verdicts"`
	Explanation string                     `json:"explanation"`
}

// Found reports whether any policy governs the stop.
func (r Resolution) Found() bool { return r.Policy != nil }

// ResolvePolicy picks the single policy that governs a stop.
//
// Precedence is Priority, then specificity, then age. Specificity is derived
// from which scope dimensions a policy pins, so a facility-specific term
// automatically beats a customer-wide one without anyone maintaining an
// ordering by hand.
func ResolvePolicy(
	candidates []*detention.DetentionPolicy,
	in detention.MatchInput,
) Resolution {
	verdicts := make([]CandidateVerdict, 0, len(candidates))
	matched := make([]*detention.DetentionPolicy, 0, len(candidates))

	for _, policy := range candidates {
		if policy == nil {
			continue
		}

		verdict := CandidateVerdict{
			PolicyID:         policy.ID,
			PolicyCode:       policy.Code,
			PolicyName:       policy.Name,
			Priority:         policy.Priority,
			SpecificityScore: policy.ComputeSpecificity(),
		}

		if policy.Matches(in) {
			verdict.Matched = true
			verdict.Reason = describeScope(policy)
			matched = append(matched, policy)
		} else {
			verdict.Reason = rejectionReason(policy, in)
		}

		verdicts = append(verdicts, verdict)
	}

	if len(matched) == 0 {
		sortVerdicts(verdicts)
		return Resolution{
			Verdicts:    verdicts,
			Explanation: "No detention policy covers this stop.",
		}
	}

	sort.SliceStable(matched, func(a, b int) bool {
		return lessSpecific(matched[b], matched[a])
	})

	winner := matched[0]

	for i := range verdicts {
		if verdicts[i].PolicyID == winner.ID {
			verdicts[i].IsWinner = true
		}
	}

	sortVerdicts(verdicts)

	return Resolution{
		Policy:      winner,
		Verdicts:    verdicts,
		Explanation: explain(winner, matched),
	}
}

// lessSpecific reports whether a ranks below b under the precedence rules.
func lessSpecific(a, b *detention.DetentionPolicy) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}

	aScore := a.ComputeSpecificity()
	bScore := b.ComputeSpecificity()
	if aScore != bScore {
		return aScore < bScore
	}

	if a.CreatedAt != b.CreatedAt {
		return a.CreatedAt > b.CreatedAt
	}

	return a.ID.String() > b.ID.String()
}

func sortVerdicts(verdicts []CandidateVerdict) {
	sort.SliceStable(verdicts, func(a, b int) bool {
		if verdicts[a].IsWinner != verdicts[b].IsWinner {
			return verdicts[a].IsWinner
		}
		if verdicts[a].Matched != verdicts[b].Matched {
			return verdicts[a].Matched
		}
		if verdicts[a].Priority != verdicts[b].Priority {
			return verdicts[a].Priority > verdicts[b].Priority
		}
		return verdicts[a].SpecificityScore > verdicts[b].SpecificityScore
	})
}

func explain(winner *detention.DetentionPolicy, matched []*detention.DetentionPolicy) string {
	scope := describeScope(winner)

	if len(matched) == 1 {
		return fmt.Sprintf("%s (%s) is the only policy covering this stop; it applies %s.",
			winner.Name, winner.Code, scope)
	}

	runnerUp := matched[1]

	return fmt.Sprintf(
		"%s (%s) won over %d other matching %s because it applies %s "+
			"(priority %d, specificity %d) while the next closest, %s (%s), applies %s "+
			"(priority %d, specificity %d).",
		winner.Name, winner.Code,
		len(matched)-1, stringutils.Pluralize("policy", "policies", len(matched)-1),
		scope, winner.Priority, winner.ComputeSpecificity(),
		runnerUp.Name, runnerUp.Code, describeScope(runnerUp),
		runnerUp.Priority, runnerUp.ComputeSpecificity(),
	)
}

func describeScope(policy *detention.DetentionPolicy) string {
	if policy.IsOrgDefault {
		return "as the organization default"
	}

	parts := make([]string, 0, 6)

	if policy.LocationID != nil && !policy.LocationID.IsNil() {
		parts = append(parts, "to a specific facility")
	}
	if policy.CustomerID != nil && !policy.CustomerID.IsNil() {
		parts = append(parts, "to a specific customer")
	}
	if len(policy.CommodityIDs) > 0 {
		parts = append(parts, "to selected commodities")
	}
	if len(policy.ServiceTypeIDs) > 0 {
		parts = append(parts, "to selected service types")
	}
	if len(policy.ShipmentTypeIDs) > 0 {
		parts = append(parts, "to selected shipment types")
	}
	if len(policy.StopTypes) > 0 {
		parts = append(parts, "to selected stop types")
	}

	if len(parts) == 0 {
		return "to all freight"
	}

	return strings.Join(parts, " and ")
}

//nolint:gocognit // one branch per scope dimension keeps the message specific
func rejectionReason(policy *detention.DetentionPolicy, in detention.MatchInput) string {
	if policy.Status != detention.PolicyStatusActive {
		return fmt.Sprintf("Policy status is %s", policy.Status)
	}

	if !policy.IsEffectiveAt(in.Timestamp) {
		return "Outside the policy's effective date range"
	}

	if policy.CustomerID != nil && !policy.CustomerID.IsNil() &&
		*policy.CustomerID != in.CustomerID {
		return "Scoped to a different customer"
	}

	if policy.LocationID != nil && !policy.LocationID.IsNil() &&
		*policy.LocationID != in.LocationID {
		return "Scoped to a different facility"
	}

	if len(policy.ShipmentTypeIDs) > 0 {
		return "Shipment type is not in scope"
	}

	if len(policy.ServiceTypeIDs) > 0 {
		return "Service type is not in scope"
	}

	if len(policy.CommodityIDs) > 0 {
		return "No shipment commodity is in scope"
	}

	if len(policy.StopTypes) > 0 {
		return "Stop type is not in scope"
	}

	if policy.AppointmentStopsOnly {
		return "Policy applies only to appointment stops"
	}

	return "Did not match"
}
