package resolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// requireTeamScope is requirePermission for an action aimed at one worker.
//
// The permission engine answers whether the grant exists and how wide it is;
// it deliberately knows nothing about who manages whom, because teaching it
// would point the authorization layer at the HR module. So the narrowing
// happens here, where the worker being acted on is already known: a grant
// scoped to "team" is checked against the people the user actually manages,
// and every wider scope is left alone.
//
// The delegator, when the answer came through one, is returned so the action
// can record that it was taken in somebody's place rather than in the actor's
// own right.
func (r *Resolver) requireTeamScope(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
	approvalScope worker.ApprovalScope,
	workerID pulid.ID,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, pulid.Nil, errortypes.NewAuthenticationError("Authentication required")
	}

	result, err := r.permissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return nil, pulid.Nil, err
	}
	if !result.Allowed {
		return nil, pulid.Nil, errortypes.NewAuthorizationError(
			fmt.Sprintf(
				"You don't have permission to perform this action: %s %s",
				resource,
				operation,
			),
		)
	}

	if result.DataScope != permission.DataScopeTeam {
		return authCtx, pulid.Nil, nil
	}

	// A team-scoped grant with no org structure service behind it would widen
	// silently, so the absence is refused rather than ignored.
	if r.orgStructureService == nil {
		return nil, pulid.Nil, errortypes.NewAuthorizationError(
			"Your access is limited to your own team, which cannot be checked right now",
		)
	}

	scope, err := r.orgStructureService.CanActFor(ctx, &orgstructureservice.ScopeRequest{
		TenantInfo: tenantInfo(authCtx),
		UserID:     authCtx.UserID,
		WorkerID:   workerID,
		Scope:      approvalScope,
	})
	if err != nil {
		return nil, pulid.Nil, err
	}
	if !scope.Allowed {
		return nil, pulid.Nil, errortypes.NewAuthorizationError(
			"That worker is not on your team",
		)
	}

	return authCtx, scope.OnBehalfOf, nil
}

// requirePTOTeamScope is requireTeamScope for a decision on one time-off
// request. The worker is looked up from the request rather than taken from the
// caller: an id supplied alongside the one being acted on is an id somebody can
// lie about.
func (r *Resolver) requirePTOTeamScope(
	ctx context.Context,
	operation permission.Operation,
	ptoID pulid.ID,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, pulid.Nil, errortypes.NewAuthenticationError("Authentication required")
	}

	pto, err := r.workerPTOService.Get(ctx, &repositories.GetPTOByIDRequest{
		ID:         ptoID,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		return nil, pulid.Nil, err
	}

	return r.requireTeamScope(
		ctx,
		permission.ResourceWorkerPTO,
		operation,
		worker.ApprovalScopeTimeOff,
		pto.WorkerID,
	)
}
