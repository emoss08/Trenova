package workeroverviewresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/pkg/authctx"
)

// overviewSections turns the signed-in user's permissions into the set of
// roll-ups the overview may compose. A section the user cannot read is left
// out entirely rather than returned empty, so a dispatcher without safety
// access sees a verdict drawn only from what they are allowed to see instead
// of one that quietly claims the safety record is clean.
func (r *Deps) OverviewSections(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) workeroverviewservice.Sections {
	return workeroverviewservice.Sections{
		Credentials: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceWorkerCredential,
			permission.OpRead,
		),
		Training: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceWorkerTraining,
			permission.OpRead,
		),
		Safety: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceWorkerSafetyEvent,
			permission.OpRead,
		),
		Checklists: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceWorkerChecklist,
			permission.OpRead,
		),
		PTO: r.HasPermission(ctx, authCtx, permission.ResourceWorkerPTO, permission.OpRead),
		Reviews: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourcePerformanceReview,
			permission.OpRead,
		),
	}
}

// AttentionAccess is what the home tile may count for the reader. Leave is
// counted across their team when their leave grant reaches only their team.
func (r *Deps) AttentionAccess(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) (workeroverviewservice.AttentionAccess, error) {
	access := workeroverviewservice.AttentionAccess{
		Reviews: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourcePerformanceReview,
			permission.OpRead,
		),
		Leave: r.HasPermission(ctx, authCtx, permission.ResourceWorkerLeave, permission.OpRead),
	}
	if !access.Leave {
		return access, nil
	}
	_, managerIDs, err := r.RequireListTeamScope(
		ctx,
		permission.ResourceWorkerLeave,
		permission.OpRead,
		approvaldelegation.ApprovalScopeTimeOff,
	)
	if err != nil {
		return access, err
	}
	access.LeaveManagerIDs = managerIDs
	return access, nil
}
