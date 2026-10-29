package agentdecisionservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	previewRefusalStale    = "preview_stale"
	previewRefusalConflict = "preview_conflict"
	previewRefusalRequired = "preview_required"
	sha256HexLength        = 64
)

// shownPreview is what a decision records of the preview its decider saw.
type shownPreview struct {
	preview       *agentpreview.ProposalPreview
	digest        string
	reviewed      bool
	targetVersion *int64
	refusal       error
}

// settlePreview works out what an approval approves: the write previewed as
// the decider would see it, with their changes, as the world is now.
//
//   - A change whose pinned record has moved on, or is gone, is refused as
//     approved: the approval is recorded with the preview it was refused on,
//     its proposal is closed as not carried out, nothing runs, and the
//     conversation that raised it is told why.
//   - A digest the decider sent that no longer matches is a conflict, and
//     nothing is recorded: a person only approves what they were shown.
//   - An approval without a digest is recorded, with the preview it ran
//     against, as not reviewed, but only for an internal or personal write
//     from a run that read nothing from outside. A change that moves money,
//     that a customer or a driver sees, that is sent outside the
//     organization, or that a tainted run proposed is refused until its
//     decider has been shown its preview.
//
// A rejection records only the digest its decider sent. A plan's step
// arrives with the preview the plan was decided on, projected from the
// steps before it, and its record was checked with the plan.
func (s *Service) settlePreview(
	ctx context.Context,
	proposal *agent.AgentProposal,
	req *agentports.DecideAgentProposalRequest,
	actor *permissionports.RequestActor,
) (*shownPreview, error) {
	// A person can always reject, whatever the preview has become since they
	// read it: the digest they saw is kept when it is one, and never checked.
	if req.Decision != agentdecision.DecisionAccepted &&
		req.Decision != agentdecision.DecisionModified {
		if !stringutils.IsLowerHexOfLength(req.PreviewDigest, sha256HexLength) {
			return &shownPreview{}, nil
		}

		return &shownPreview{digest: req.PreviewDigest}, nil
	}

	needsReview := s.needsReview(proposal, req.Modifications)

	if req.StepPreview != nil {
		reviewed := req.StepPreviewReviewed && req.StepPreview.Digest != ""
		refusal := staleRefusal(proposal, req.StepPreview)
		if needsReview && !reviewed && refusal == nil {
			return nil, agent.ReviewRequiredError()
		}

		return &shownPreview{
			preview:       req.StepPreview,
			digest:        req.StepPreview.Digest,
			reviewed:      reviewed,
			targetVersion: req.StepPreview.TargetVersion,
			refusal:       refusal,
		}, nil
	}

	// A step of a plan decided without a plan preview is not previewed on its
	// own: after the steps before it have run, its record has moved by design,
	// and the plan settled what was approved.
	if s.previews == nil || req.WithinPlan {
		if needsReview {
			return nil, agent.ReviewRequiredError()
		}

		return &shownPreview{}, nil
	}
	if needsReview && req.PreviewDigest == "" {
		return nil, agent.ReviewRequiredError()
	}

	previewReq := &agentruntimeports.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   &agentruntimeports.PreviewViewer{Actor: actor},
		Deciding: true,
	}
	if len(req.Modifications) > 0 {
		previewReq.Params = proposalexecutor.MergeParams(proposal.ToolParams, req.Modifications)
	}

	preview, err := s.previews.ForProposal(ctx, previewReq)
	if err != nil {
		return nil, err
	}

	if refusal := staleRefusal(proposal, preview); refusal != nil {
		return &shownPreview{
			preview:       preview,
			digest:        preview.Digest,
			targetVersion: preview.TargetVersion,
			refusal:       refusal,
		}, nil
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

func (s *Service) needsReview(proposal *agent.AgentProposal, modifications map[string]any) bool {
	if s.executor == nil {
		return proposal.ApprovalNeedsReview() || len(modifications) > 0
	}

	return s.executor.ReviewRequired(proposal, modifications)
}

func staleRefusal(proposal *agent.AgentProposal, preview *agentpreview.ProposalPreview) error {
	if !preview.IsStale() {
		return nil
	}

	record := targetName(proposal, preview)
	if preview.Staleness.Missing {
		return fmt.Errorf(
			"%w: %s no longer exists, so nothing was changed",
			agentplan.ErrTargetChanged, record,
		)
	}

	return fmt.Errorf(
		"%w: %s was at version %d when this was proposed and is at version %d now, "+
			"so nothing was changed",
		agentplan.ErrTargetChanged,
		record,
		preview.Staleness.ProposedVersion,
		preview.Staleness.CurrentVersion,
	)
}

func targetName(proposal *agent.AgentProposal, preview *agentpreview.ProposalPreview) string {
	noun := "the record"
	if resource := strings.TrimSpace(proposal.TargetResource); resource != "" {
		noun = "the " + strings.ReplaceAll(resource, "_", " ")
	}

	for i := range preview.Changes {
		change := &preview.Changes[i]
		if change.EntityID == proposal.TargetID && proposal.TargetID.IsNotNil() &&
			strings.TrimSpace(change.Label) != "" {
			return noun + " " + strings.TrimSpace(change.Label)
		}
	}

	return noun
}

func previewRefusal(err error) string {
	if agent.IsReviewRequired(err) {
		return previewRefusalRequired
	}

	var conflict *errortypes.ConflictError
	if errors.As(err, &conflict) {
		return previewRefusalConflict
	}

	return aitrace.OutcomeFailed
}
