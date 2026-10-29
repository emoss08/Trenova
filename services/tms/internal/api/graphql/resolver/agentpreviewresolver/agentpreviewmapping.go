package agentpreviewresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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
