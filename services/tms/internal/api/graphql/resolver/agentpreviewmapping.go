package resolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// previewViewer is the reader a preview is filtered for, with the request's
// own caches: the ceiling per resource from the loaders and every read check
// through the request's permission memo, so a page asks the engine once per
// resource.
func (r *Resolver) previewViewer(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) *services.PreviewViewer {
	viewer := &services.PreviewViewer{
		Actor: actorutil.FromAuthContext(authCtx),
		Reads: requestReadAccess{resolver: r, authCtx: authCtx},
	}
	if l, ok := loaders.FromContext(ctx); ok && l != nil {
		viewer.Ceilings = l.FieldCeilings
	}

	return viewer
}

// requestReadAccess answers read checks through the request's memoised
// permission check.
type requestReadAccess struct {
	resolver *Resolver
	authCtx  *authctx.AuthContext
}

func (a requestReadAccess) MayRead(ctx context.Context, resource permission.Resource) bool {
	return a.resolver.hasPermission(ctx, a.authCtx, resource, permission.OpRead)
}

func (r *Resolver) previewService() (services.ProposalPreviewService, error) {
	if r.proposalPreviewService == nil {
		return nil, errortypes.NewBusinessError("Previews of proposed changes are not available")
	}

	return r.proposalPreviewService, nil
}

func (r *Resolver) proposalPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	proposalID pulid.ID,
	modifications map[string]any,
) (*agent.ProposalPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	tenant := tenantInfo(authCtx)
	proposal, err := r.agentProposalService.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         proposalID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}

	return previews.ForProposal(ctx, &services.ProposalPreviewRequest{
		Proposal:      proposal,
		Modifications: modifications,
		Viewer:        r.previewViewer(ctx, authCtx),
	})
}

func (r *Resolver) planPreview(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	planID pulid.ID,
) (*agent.PlanPreview, error) {
	previews, err := r.previewService()
	if err != nil {
		return nil, err
	}

	plan, err := r.agentPlanService.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		return nil, err
	}

	return previews.ForPlan(ctx, &services.PlanPreviewRequest{
		Plan:   plan,
		Viewer: r.previewViewer(ctx, authCtx),
	})
}

// previewDigestsByProposal reads a batch's digests, one per proposal.
func previewDigestsByProposal(
	inputs []*gqlmodel.AgentProposalPreviewDigestInput,
) (map[pulid.ID]string, error) {
	digests := make(map[pulid.ID]string, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		id, err := pulid.Parse(input.ProposalID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("previewDigests[%d].proposalId", i),
				errortypes.ErrInvalid,
				"Proposal identifier is invalid",
			)
		}
		if _, dup := digests[id]; dup {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("previewDigests[%d].proposalId", i),
				errortypes.ErrInvalid,
				"A proposal's digest is listed twice",
			)
		}
		digests[id] = input.Digest
	}

	return digests, nil
}
