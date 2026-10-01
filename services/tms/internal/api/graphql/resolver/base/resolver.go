package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type Core struct {
	L                   *zap.Logger
	WorkerPTOService    services.WorkerPTOService
	OrgStructureService *orgstructureservice.Service
	PermissionEngine    services.PermissionEngine
	TraceURL            func(traceID string) string
}

func (r *Core) RequirePermission(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	allowed, err := r.checkPermission(ctx, authCtx, resource, operation)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}", resource, operation,
		)
	}

	return authCtx, nil
}

func (r *Core) HasPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource permission.Resource,
	operation permission.Operation,
) bool {
	allowed, err := r.checkPermission(ctx, authCtx, resource, operation)
	if err != nil {
		r.L.Warn("permission check failed",
			zap.String("resource", resource.String()),
			zap.Error(err))
		return false
	}

	return allowed
}

func (r *Core) checkPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource permission.Resource,
	operation permission.Operation,
) (bool, error) {
	memo, memoised := gqlctx.PermissionMemoFrom(ctx)
	key := resource.String() + "|" + string(operation)
	if memoised {
		if allowed, found := memo.Lookup(key); found {
			return allowed, nil
		}
	}

	result, err := r.PermissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return false, err
	}
	if memoised {
		memo.Store(key, result.Allowed)
	}

	return result.Allowed, nil
}

func (r *Core) RequireAuth(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

func TenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
