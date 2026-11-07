package auditlogresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func narrowed(scope permission.DataScope) bool {
	return scope == permission.DataScopeOwn || scope == permission.DataScopeTeam
}

func (r *QueryResolver) authContext(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}
	return authCtx, nil
}

func (r *Deps) readableResources(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) (auditports.EntryAccess, error) {
	defs := permissioncatalog.Default().All()
	checks := make([]permissionports.ResourceOperationCheck, 0, len(defs)+1)
	for _, def := range defs {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  def.Resource,
			Operation: permission.OpRead,
		})
	}
	checks = append(checks, permissionports.ResourceOperationCheck{
		Resource:  permission.ResourceSecurityAuditLog.String(),
		Operation: permission.OpRead,
	})

	single := middleware.BuildPermissionCheckRequest(authCtx, "", permission.OpRead)
	result, err := r.PermissionEngine.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:     single.PrincipalType,
		PrincipalID:       single.PrincipalID,
		UserID:            single.UserID,
		APIKeyID:          single.APIKeyID,
		BusinessUnitID:    single.BusinessUnitID,
		OrganizationID:    single.OrganizationID,
		Checks:            checks,
		ContextAttributes: single.ContextAttributes,
	})
	if err != nil {
		return auditports.EntryAccess{}, err
	}
	if len(result.Results) != len(checks) {
		return auditports.EntryAccess{}, errortypes.NewBusinessError(
			"Your permissions could not be read; try again",
		)
	}

	access := auditports.EntryAccess{
		IncludeSecurity: result.Results[len(checks)-1].Allowed,
	}
	all := true
	for i, def := range defs {
		res := result.Results[i]
		if res.Allowed && !narrowed(res.DataScope) {
			access.Resources = append(access.Resources, permission.Resource(def.Resource))
			continue
		}
		all = false
	}
	if all {
		access.AllResources = true
		access.Resources = nil
	}
	return access, nil
}

func (r *Deps) recordAccess(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resourceID pulid.ID,
) (auditports.EntryAccess, error) {
	includeSecurity := r.HasPermission(
		ctx,
		authCtx,
		permission.ResourceSecurityAuditLog,
		permission.OpRead,
	)

	resource, known := recordkind.ResourceOfIDPrefix(resourceID.Prefix())
	if !known {
		if !r.HasPermission(ctx, authCtx, permission.ResourceAuditLog, permission.OpRead) {
			return auditports.EntryAccess{}, errortypes.NewAuthorizationError(
				"You don't have permission to read this record's history",
			)
		}
		access, err := r.readableResources(ctx, authCtx)
		if err != nil {
			return auditports.EntryAccess{}, err
		}
		return access, nil
	}

	result, err := r.PermissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), permission.OpRead),
	)
	if err != nil {
		return auditports.EntryAccess{}, err
	}
	if result == nil || !result.Allowed {
		return auditports.EntryAccess{}, errortypes.NewAuthorizationError(
			"You don't have permission to read this record's history",
		)
	}
	if narrowed(result.DataScope) {
		return auditports.EntryAccess{}, errortypes.NewAuthorizationError(
			"Your access to these records is limited to your own or your team's, so their history is not shown",
		)
	}

	return auditports.EntryAccess{
		Resources:       []permission.Resource{resource},
		IncludeSecurity: includeSecurity,
	}, nil
}
