package agentplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// shownPlan is the plan preview an approval approved, and whether its
// decider named its digest.
type shownPlan struct {
	preview  *agent.PlanPreview
	reviewed bool
}

// annotate hands a step the preview the plan was approved on, so its
// decision records the step as the decider saw it, projected from the steps
// before it, rather than as it reads once they have run.
func (p *shownPlan) annotate(req *services.DecideAgentProposalRequest, proposalID pulid.ID) {
	if p == nil || p.preview == nil {
		return
	}

	req.StepPreview = p.preview.StepFor(proposalID)
	req.StepPreviewReviewed = p.reviewed
}

// settlePreview works out what approving a plan approves: every pending
// step previewed as the decider would see it, each projected from the steps
// before it. A plan with a step whose record moved on is refused, and a plan
// digest that no longer matches is a conflict; either way nothing is
// recorded and nothing runs. A rejection needs no preview.
func (s *Service) settlePreview(
	ctx context.Context,
	req *services.DecideAgentPlanRequest,
	plan *agent.AgentPlan,
	steps []*agent.AgentProposal,
	actor *services.RequestActor,
) (*shownPlan, error) {
	if req.Decision != agent.DecisionAccepted || s.previews == nil {
		return nil, nil //nolint:nilnil // no preview settles a rejection
	}

	preview, err := s.previews.ForPlan(ctx, &services.PlanPreviewRequest{
		Plan:   plan,
		Steps:  steps,
		Viewer: &services.PreviewViewer{Actor: actor},
	})
	if err != nil {
		return nil, err
	}

	if preview.Stale {
		return nil, errortypes.NewBusinessError(
			"A step of this plan changes a record that has changed since the plan was " +
				"proposed, so it cannot be approved as it stands. Reject it, or ask the agent " +
				"again for a current plan",
		)
	}

	if req.PreviewDigest != "" && req.PreviewDigest != preview.Digest {
		s.metrics.RecordConflict(agent.PlanToolName)

		return nil, errortypes.NewConflictError(
			"What this plan would do has changed since you reviewed it. " +
				"Review it again, then decide",
		)
	}

	return &shownPlan{preview: preview, reviewed: req.PreviewDigest != ""}, nil
}
