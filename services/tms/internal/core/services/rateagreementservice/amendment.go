package rateagreementservice

import (
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// planRuleAmendment turns the lane set a save submitted into the amendment
// that reconciles the stored contract with it.
//
// A save is somebody's whole picture of the lanes, so the plan is a diff
// against what the contract holds now: a submitted lane without an identity is
// new; one whose identity is known and whose terms moved supersedes its
// predecessor and starts a successor at the amendment moment; one restated
// identically is left entirely alone, which is what keeps the history free of
// amendments that amended nothing; and a stored lane the save no longer names
// is closed out. Nothing is ever edited in place — that is the amend
// machinery's contract, and this planner only decides what to feed it.
//
// amendAt is the instant the change takes effect: superseded lanes stop
// pricing there and their successors begin there, so no lane is ever priced
// twice or not at all across the boundary.
func planRuleAmendment(
	existing []*rateagreement.RateAgreementRule,
	submitted []*rateagreement.RateAgreementRule,
	amendAt int64,
) (*ruleAmendmentPlan, *errortypes.MultiError) {
	multiErr := errortypes.NewMultiError()
	plan := &ruleAmendmentPlan{}

	existingByID := make(map[pulid.ID]*rateagreement.RateAgreementRule, len(existing))
	for _, rule := range existing {
		if rule != nil && !rule.ID.IsNil() {
			existingByID[rule.ID] = rule
		}
	}

	kept := make(map[pulid.ID]struct{}, len(submitted))

	for index, rule := range submitted {
		if rule == nil {
			continue
		}

		if rule.ID.IsNil() {
			plan.Inserts = append(plan.Inserts, rule)
			continue
		}

		prior, known := existingByID[rule.ID]
		if !known {
			// An identity the contract does not hold — a stale row, or one
			// pasted in from elsewhere. It cannot supersede anything, so it
			// lands as a new lane; the insert path mints it a fresh identity.
			plan.Inserts = append(plan.Inserts, rule)
			continue
		}

		kept[rule.ID] = struct{}{}

		if prior.SameTerms(rule) {
			continue
		}

		if rule.EffectiveTo != nil && *rule.EffectiveTo <= amendAt {
			multiErr.WithIndex("rules", index).Add(
				"effectiveTo",
				errortypes.ErrInvalid,
				"This lane's window closes before the change would take effect",
			)
			continue
		}

		supersededID := rule.ID
		rule.SupersedesRuleID = &supersededID
		rule.EffectiveFrom = amendAt

		plan.SupersededIDs = append(plan.SupersededIDs, supersededID)
		plan.Inserts = append(plan.Inserts, rule)
	}

	for _, rule := range existing {
		if rule == nil || rule.ID.IsNil() {
			continue
		}

		if _, stillNamed := kept[rule.ID]; !stillNamed {
			plan.SupersededIDs = append(plan.SupersededIDs, rule.ID)
		}
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if plan.Empty() {
		return nil, nil
	}

	return plan, nil
}

type ruleAmendmentPlan struct {
	SupersededIDs []pulid.ID
	Inserts       []*rateagreement.RateAgreementRule
}

func (p *ruleAmendmentPlan) Empty() bool {
	return len(p.SupersededIDs) == 0 && len(p.Inserts) == 0
}
