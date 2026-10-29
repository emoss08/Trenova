package permission

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (e *engine) checkAgentPermission(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
	start time.Time,
	policies accessPolicySource,
) (*permissionports.PermissionCheckResult, error) {
	result := &permissionports.PermissionCheckResult{
		Allowed:   agentceiling.Allows(permission.Resource(req.Resource), req.Operation),
		Reason:    "agent_denied",
		DataScope: permission.DataScopeOrganization,
	}
	if result.Allowed {
		result.Reason = "agent_allowed"

		role, err := e.agentRolePermissions(ctx, req)
		if err != nil {
			return nil, err
		}
		if role != nil {
			if _, ok := e.resolveResourcePermission(role, req.Resource, req.Operation); !ok {
				result.Allowed = false
				result.Reason = "agent_role_denied"
			}
		}
	}
	result.CheckDuration = time.Since(start).Milliseconds()

	if !result.Allowed {
		return result, nil
	}
	return e.applyAccessPolicies(ctx, req, result, policies)
}

func (e *engine) checkAgentBatch(
	ctx context.Context,
	req *permissionports.BatchPermissionCheckRequest,
	results []permissionports.PermissionCheckResult,
) error {
	policies := e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID)
	for i, check := range req.Checks {
		result, err := e.checkAgentPermission(
			ctx,
			batchCheckRequest(req, check),
			time.Now(),
			policies,
		)
		if err != nil {
			return err
		}
		results[i] = *result
	}
	return nil
}

func (e *engine) agentRolePermissions(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.CachedPermissions, error) {
	if e.agentRoles == nil || req.PrincipalID.IsNil() ||
		req.PrincipalID == permissionports.AgentPrincipalID {
		return nil, nil //nolint:nilnil // an agent with no definition holds no role
	}

	key := permissionports.PermissionCacheKey{
		UserID:  req.PrincipalID,
		OrgID:   req.OrganizationID,
		Variant: permissionports.PermissionCacheVariantAgentRole,
	}
	cached, err := e.cacheRepo.Get(ctx, key)
	if err != nil {
		e.l.Warn("agent role cache lookup failed, computing fresh", zap.Error(err))
	}
	if err == nil && cached != nil && cached.ExpiresAt > timeutils.NowUnix() {
		return agentRoleOrNil(cached), nil
	}

	epoch, epochErr := e.cacheRepo.Epoch(ctx, req.OrganizationID)
	perms, err := e.computeAgentRolePermissions(ctx, req)
	if err != nil {
		return nil, err
	}
	if epochErr == nil {
		perms.Epoch = epoch
		if cacheErr := e.cacheRepo.Set(ctx, key, perms, cacheTTL); cacheErr != nil {
			e.l.Warn("failed to cache agent role permissions", zap.Error(cacheErr))
		}
	}

	return agentRoleOrNil(perms), nil
}

func agentRoleOrNil(perms *permissionports.CachedPermissions) *permissionports.CachedPermissions {
	if perms == nil || perms.AgentRoleID == "" {
		return nil
	}
	return perms
}

func (e *engine) computeAgentRolePermissions(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.CachedPermissions, error) {
	perms := &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityPublic),
		Resources:      make(map[string]*permissionports.CachedResourcePermission),
		ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}

	roleID, err := e.agentRoles.UnattendedRoleID(ctx, pagination.TenantInfo{
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
	}, req.PrincipalID)
	if err != nil {
		return nil, fmt.Errorf("read the role an agent runs as: %w", err)
	}
	if roleID.IsNil() {
		return perms, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, []pulid.ID{roleID})
	if err != nil {
		return nil, fmt.Errorf("read the role an agent runs as: %w", err)
	}

	maxSensitivity := permission.SensitivityPublic
	for _, role := range roles {
		applyRoleSensitivity(role, &maxSensitivity)
		e.mergeRolePermissionsIntoCache(perms.Resources, role.Permissions)
	}
	perms.MaxSensitivity = string(maxSensitivity)
	perms.AgentRoleID = roleID.String()

	return perms, nil
}
