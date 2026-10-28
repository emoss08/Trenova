package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
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
func (r *Core) RequireTeamScope(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
	approvalScope approvaldelegation.ApprovalScope,
	workerID pulid.ID,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, pulid.Nil, errortypes.NewAuthenticationError("Authentication required")
	}

	result, err := r.PermissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return nil, pulid.Nil, err
	}
	if !result.Allowed {
		return nil, pulid.Nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}", resource, operation,
		)
	}

	if result.DataScope != permission.DataScopeTeam {
		return authCtx, pulid.Nil, nil
	}

	// A team-scoped grant with no org structure service behind it would widen
	// silently, so the absence is refused rather than ignored.
	if r.OrgStructureService == nil {
		return nil, pulid.Nil, errortypes.NewAuthorizationError(
			"Your access is limited to your own team, which cannot be checked right now",
		)
	}

	scope, err := r.OrgStructureService.CanActFor(ctx, &orgstructureservice.ScopeRequest{
		TenantInfo: TenantInfo(authCtx),
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
func (r *Core) RequirePTOTeamScope(
	ctx context.Context,
	operation permission.Operation,
	ptoID pulid.ID,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, pulid.Nil, errortypes.NewAuthenticationError("Authentication required")
	}

	pto, err := r.WorkerPTOService.Get(ctx, &workerports.GetPTOByIDRequest{
		ID:         ptoID,
		TenantInfo: TenantInfo(authCtx),
	})
	if err != nil {
		return nil, pulid.Nil, err
	}

	return r.RequireTeamScope(
		ctx,
		permission.ResourceWorkerPTO,
		operation,
		approvaldelegation.ApprovalScopeTimeOff,
		pto.WorkerID,
	)
}

// RequireListTeamScope is requirePermission for a list that spans workers. A
// grant scoped to "team" narrows the list to the people the user answers for,
// directly or through a fleet code, and the managers it returns are what the
// repository filters by; a wider grant returns none and the list is the
// organization's. A team-scoped caller who answers for no one, such as an
// unattended agent, which names no person, is refused: no managers would
// read as the whole organization.
func (r *Core) RequireListTeamScope(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
	approvalScope approvaldelegation.ApprovalScope,
) (*authctx.AuthContext, []pulid.ID, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, nil, errortypes.NewAuthenticationError("Authentication required")
	}

	result, err := r.PermissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return nil, nil, err
	}
	if !result.Allowed {
		return nil, nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}", resource, operation,
		)
	}
	if result.DataScope != permission.DataScopeTeam {
		return authCtx, nil, nil
	}

	if r.OrgStructureService == nil {
		return nil, nil, errortypes.NewAuthorizationError(
			"Your access is limited to your own team, which cannot be checked right now",
		)
	}
	managerIDs, _, err := r.OrgStructureService.ManagerIDs(
		ctx,
		TenantInfo(authCtx),
		authCtx.UserID,
		approvalScope,
		0,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(managerIDs) == 0 {
		return nil, nil, errortypes.NewAuthorizationError(
			"Your access is limited to your own team, and you answer for no one",
		)
	}

	return authCtx, managerIDs, nil
}
