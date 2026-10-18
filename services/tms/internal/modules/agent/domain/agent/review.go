package agent

import (
	"errors"
	"slices"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const reviewRequiredDetail = "preview_required"

func (p *AgentProposal) ApprovalNeedsReview() bool {
	return p.Tainted || p.EgressClass.ApprovalNeedsReview()
}

func PlanNeedsReview(steps []*AgentProposal) bool {
	return slices.ContainsFunc(steps, func(step *AgentProposal) bool {
		return step != nil && step.Status == ProposalStatusPending && step.ApprovalNeedsReview()
	})
}

func ReviewRequiredError() error {
	refusal := errortypes.NewBusinessError(
		"This change must be reviewed before it is approved. Open its preview, check what " +
			"it would do, then approve it",
	)
	refusal.Details = reviewRequiredDetail

	return refusal
}

func IsReviewRequired(err error) bool {
	var business *errortypes.BusinessError

	return errors.As(err, &business) && business.Details == reviewRequiredDetail
}
