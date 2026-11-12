package agentpreviewresolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// previewViewer is the reader a preview is filtered for. The ceiling per
// resource comes from the request's loaders, and read access is checked for
// the reader as an actor, exactly as a decision checks it, so the digest a
// person is shown is the digest their approval is checked against.
func (r *Deps) PreviewViewer(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) *agentruntimeports.PreviewViewer {
	actor := actorutil.FromAuthContext(authCtx)
	viewer := &agentruntimeports.PreviewViewer{
		Actor: actor,
		Reads: fieldsensitivity.NewReadAccess(r.PermissionEngine, actor),
	}
	viewer.Ceilings = agentloaders.FieldCeilings(ctx)

	return viewer
}

func (r *Deps) previewService() (agentruntimeports.ProposalPreviewService, error) {
	if r.ProposalPreviewService == nil {
		return nil, errortypes.NewBusinessError("Previews of proposed changes are not available")
	}

	return r.ProposalPreviewService, nil
}

func (r *Deps) ProposalPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	proposalID pulid.ID,
	modifications map[string]any,
) (*agentpreview.ProposalPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	tenant := resolvercore.TenantInfo(authCtx)
	proposal, err := r.AgentProposalService.GetByID(
		ctx,
		agentproposalports.GetAgentProposalByIDRequest{
			ID:         proposalID,
			TenantInfo: &tenant,
		},
	)
	if err != nil {
		return nil, err
	}

	return previews.ForProposal(ctx, &agentruntimeports.ProposalPreviewRequest{
		Proposal:      proposal,
		Modifications: modifications,
		Viewer:        r.PreviewViewer(ctx, authCtx),
	})
}

// ProposalPreviews previews up to agentdecision.MaxBatch proposals in the
// order asked, for deciding them together. The proposals are read at once;
// one that is missing or fails to preview carries its error, and an
// internal failure is logged here because its reader is told only that the
// change could not be previewed.
func (r *Deps) ProposalPreviews(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	ids []pulid.ID,
) ([]*agentruntimeports.ProposalPreviewOutcome, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}
	if err = validatePreviewBatch(ids); err != nil {
		return nil, err
	}

	proposals, err := r.AgentProposalService.ListByIDs(
		ctx,
		agentproposalports.ListAgentProposalsByIDsRequest{
			IDs:        ids,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
	)
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*agent.AgentProposal, len(proposals))
	for _, proposal := range proposals {
		byID[proposal.ID] = proposal
	}

	found := make([]*agent.AgentProposal, 0, len(ids))
	for _, id := range ids {
		if proposal, ok := byID[id]; ok {
			found = append(found, proposal)
		}
	}
	previewed := previews.ForProposals(ctx, &agentruntimeports.ProposalPreviewsRequest{
		Proposals: found,
		Viewer:    r.PreviewViewer(ctx, authCtx),
	})
	byProposal := make(map[pulid.ID]*agentruntimeports.ProposalPreviewOutcome, len(previewed))
	for i := range previewed {
		byProposal[previewed[i].ProposalID] = &previewed[i]
	}

	out := make([]*agentruntimeports.ProposalPreviewOutcome, 0, len(ids))
	for _, id := range ids {
		outcome, ok := byProposal[id]
		if !ok {
			outcome = &agentruntimeports.ProposalPreviewOutcome{
				ProposalID: id,
				Err:        errortypes.NewNotFoundError("Proposal not found"),
			}
		}
		if outcome.Err != nil && !errortypes.IsForPerson(outcome.Err) {
			r.L.Warn("a proposal of a batch could not be previewed",
				zap.String("proposal", id.String()),
				zap.Error(outcome.Err),
			)
		}
		out = append(out, outcome)
	}

	return out, nil
}

func validatePreviewBatch(ids []pulid.ID) error {
	multiErr := errortypes.NewMultiError()
	switch {
	case len(ids) == 0:
		multiErr.Add("ids", errortypes.ErrRequired, "Choose at least one proposal")
	case len(ids) > agentdecision.MaxBatch:
		multiErr.Add(
			"ids",
			errortypes.ErrInvalid,
			"Preview at most {0} proposals at once",
			agentdecision.MaxBatch,
		)
	}
	seen := make(map[pulid.ID]struct{}, len(ids))
	for i, id := range ids {
		if _, dup := seen[id]; dup {
			multiErr.Add(fmt.Sprintf("ids[%d]", i), errortypes.ErrInvalid, "Proposal listed twice")
		}
		seen[id] = struct{}{}
	}

	return multiErr.Err()
}

func (r *Deps) PlanPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	planID pulid.ID,
) (*agentpreview.PlanPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	plan, err := r.AgentPlanService.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
	if err != nil {
		return nil, err
	}

	return previews.ForPlan(ctx, &agentruntimeports.PlanPreviewRequest{
		Plan:   plan,
		Viewer: r.PreviewViewer(ctx, authCtx),
	})
}
