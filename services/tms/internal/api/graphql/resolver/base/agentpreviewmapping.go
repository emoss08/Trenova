package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// previewViewer is the reader a preview is filtered for. The ceiling per
// resource comes from the request's loaders, and read access is checked for
// the reader as an actor, exactly as a decision checks it, so the digest a
// person is shown is the digest their approval is checked against.
func (r *Resolver) PreviewViewer(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) *services.PreviewViewer {
	actor := actorutil.FromAuthContext(authCtx)
	viewer := &services.PreviewViewer{
		Actor: actor,
		Reads: fieldsensitivity.NewReadAccess(r.PermissionEngine, actor),
	}
	if l, ok := loaders.FromContext(ctx); ok && l != nil {
		viewer.Ceilings = l.FieldCeilings
	}

	return viewer
}

func (r *Resolver) previewService() (services.ProposalPreviewService, error) {
	if r.ProposalPreviewService == nil {
		return nil, errortypes.NewBusinessError("Previews of proposed changes are not available")
	}

	return r.ProposalPreviewService, nil
}

func (r *Resolver) ProposalPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	proposalID pulid.ID,
	modifications map[string]any,
) (*agent.ProposalPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	tenant := TenantInfo(authCtx)
	proposal, err := r.AgentProposalService.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         proposalID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}

	return previews.ForProposal(ctx, &services.ProposalPreviewRequest{
		Proposal:      proposal,
		Modifications: modifications,
		Viewer:        r.PreviewViewer(ctx, authCtx),
	})
}

func (r *Resolver) PlanPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	planID pulid.ID,
) (*agent.PlanPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	plan, err := r.AgentPlanService.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: TenantInfo(authCtx),
	})
	if err != nil {
		return nil, err
	}

	return previews.ForPlan(ctx, &services.PlanPreviewRequest{
		Plan:   plan,
		Viewer: r.PreviewViewer(ctx, authCtx),
	})
}
