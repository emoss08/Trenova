package agentdecisionservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	previewRefusalStale    = "preview_stale"
	previewRefusalConflict = "preview_conflict"
	sha256HexLength        = 64
)

// shownPreview is what a decision records of the preview its decider saw.
type shownPreview struct {
	preview       *agent.ProposalPreview
	digest        string
	reviewed      bool
	targetVersion *int64
}

// settlePreview works out what an approval approves: the write previewed as
// the decider would see it, with their changes, as the world is now.
//
//   - A change whose pinned record has moved on, or is gone, is refused
//     before anything is recorded. The executor would refuse it anyway, after
//     the decision said "approved"; refusing here keeps the record honest.
//   - A digest the decider sent that no longer matches is a conflict, and
//     nothing is recorded: a person only approves what they were shown.
//   - An approval without a digest is recorded, with the preview it ran
//     against, as not reviewed.
//
// A rejection records only the digest its decider sent. A plan's step
// arrives with the preview the plan was decided on, projected from the
// steps before it, and its record was checked with the plan.
func (s *Service) settlePreview(
	ctx context.Context,
	proposal *agent.AgentProposal,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*shownPreview, error) {
	// A person can always reject, whatever the preview has become since they
	// read it: the digest they saw is kept when it is one, and never checked.
	if req.Decision != agent.DecisionAccepted && req.Decision != agent.DecisionModified {
		if !stringutils.IsLowerHexOfLength(req.PreviewDigest, sha256HexLength) {
			return &shownPreview{}, nil
		}

		return &shownPreview{digest: req.PreviewDigest}, nil
	}

	if req.StepPreview != nil {
		return &shownPreview{
			preview:       req.StepPreview,
			digest:        req.StepPreview.Digest,
			reviewed:      req.StepPreviewReviewed && req.StepPreview.Digest != "",
			targetVersion: req.StepPreview.TargetVersion,
		}, nil
	}

	// A step of a plan decided without a plan preview is not previewed on its
	// own: after the steps before it have run, its record has moved by design,
	// and the plan settled what was approved.
	if s.previews == nil || req.WithinPlan {
		return &shownPreview{}, nil
	}

	previewReq := &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   &services.PreviewViewer{Actor: actor},
		Deciding: true,
	}
	if len(req.Modifications) > 0 {
		previewReq.Params = proposalexecutor.MergeParams(proposal.ToolParams, req.Modifications)
	}

	preview, err := s.previews.ForProposal(ctx, previewReq)
	if err != nil {
		return nil, err
	}

	if preview.IsStale() {
		return nil, errortypes.NewBusinessError(
			"The record this change is for has changed since it was proposed, so it cannot be " +
				"approved as it stands. Reject it, or ask the agent again for a current one",
		)
	}

	if req.PreviewDigest != "" && req.PreviewDigest != preview.Digest {
		s.metrics.RecordConflict(proposal.ToolName)

		return nil, errortypes.NewConflictError(
			"What this change would do has changed since you reviewed it. " +
				"Review it again, then decide",
		)
	}

	return &shownPreview{
		preview:       preview,
		digest:        preview.Digest,
		reviewed:      req.PreviewDigest != "",
		targetVersion: preview.TargetVersion,
	}, nil
}

func previewRefusal(err error) string {
	var conflict *errortypes.ConflictError
	if errors.As(err, &conflict) {
		return previewRefusalConflict
	}

	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return previewRefusalStale
	}

	return aitrace.OutcomeFailed
}
