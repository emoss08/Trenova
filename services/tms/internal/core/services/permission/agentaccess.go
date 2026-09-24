package permission

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

func (e *engine) grantedAgentIDs(
	ctx context.Context,
	orgID pulid.ID,
	roles []*permission.Role,
) ([]string, error) {
	if e.agentGrants == nil || len(roles) == 0 {
		return []string{}, nil
	}

	roleIDs := make([]pulid.ID, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			roleIDs = append(roleIDs, role.ID)
		}
	}

	granted, err := e.agentGrants.ListGrantedAgentIDs(ctx, repositories.ListGrantedAgentIDsRequest{
		OrganizationID: orgID,
		RoleIDs:        roleIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("read the agents granted to the roles: %w", err)
	}

	ids := make([]string, 0, len(granted))
	for _, id := range granted {
		ids = append(ids, id.String())
	}
	slices.Sort(ids)

	return slices.Compact(ids), nil
}

func (e *engine) AgentsUsable(
	ctx context.Context,
	actor *services.RequestActor,
	operation permission.Operation,
) (*services.UsableAgents, error) {
	if actor == nil {
		return &services.UsableAgents{}, nil
	}

	req := actor.PermissionCheck(permission.ResourceAssistant, operation)
	if actor.IsAPIKey() || actor.IsAgent() {
		result, err := e.Check(ctx, req)
		if err != nil {
			return nil, err
		}

		return &services.UsableAgents{Assistant: result.Allowed}, nil
	}

	load, err := e.getOrComputePermissions(ctx, actor.UserID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}

	result, err := e.checkUserPermission(
		ctx,
		e.checkLogger("AgentsUsable", req),
		time.Now(),
		load,
		req,
		e.newAccessPolicySource(actor.OrganizationID, actor.BusinessUnitID),
	)
	if err != nil {
		return nil, err
	}

	usable := &services.UsableAgents{Assistant: result.Allowed}
	if !result.Allowed {
		return usable, nil
	}

	usable.GrantedIDs = make([]pulid.ID, 0, len(load.perms.AgentIDs))
	for _, id := range load.perms.AgentIDs {
		usable.GrantedIDs = append(usable.GrantedIDs, pulid.ID(id))
	}

	return usable, nil
}

func (e *engine) MayUseAgent(
	ctx context.Context,
	actor *services.RequestActor,
	definition *agentdefinition.Definition,
) (bool, error) {
	if actor == nil || definition == nil {
		return false, nil
	}

	usable, err := e.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		return false, err
	}

	return usable.Allows(definition), nil
}

func (e *engine) RoleCoverage(
	ctx context.Context,
	req *services.RoleCoverageRequest,
) ([]services.RoleCoverage, error) {
	if req == nil || len(req.RoleIDs) == 0 {
		return []services.RoleCoverage{}, nil
	}

	closure, err := e.roleRepo.GetRolesWithInheritance(ctx, req.RoleIDs)
	if err != nil {
		return nil, fmt.Errorf("read the roles and the roles they inherit: %w", err)
	}

	byID := make(map[pulid.ID]*permission.Role, len(closure))
	for _, role := range closure {
		if role != nil && role.OrganizationID == req.OrganizationID {
			byID[role.ID] = role
		}
	}

	coverage := make([]services.RoleCoverage, 0, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		role, ok := byID[roleID]
		if !ok {
			continue
		}
		coverage = append(coverage, e.coverageOf(role, byID, req.Required))
	}

	return coverage, nil
}

func (e *engine) coverageOf(
	role *permission.Role,
	byID map[pulid.ID]*permission.Role,
	required []services.RequiredGrant,
) services.RoleCoverage {
	perms := e.inheritedPermissions(role, byID)
	result := services.RoleCoverage{
		RoleID:           role.ID,
		Coverage:         agentdefinition.CoverageNone,
		MissingResources: []permission.Resource{},
	}
	if _, ok := e.resolveResourcePermission(
		perms,
		permission.ResourceAssistant.String(),
		permission.OpCreate,
	); !ok {
		return result
	}

	for _, grant := range required {
		if _, ok := e.resolveResourcePermission(
			perms,
			grant.Resource.String(),
			grant.Operation,
		); ok {
			continue
		}
		if !slices.Contains(result.MissingResources, grant.Resource) {
			result.MissingResources = append(result.MissingResources, grant.Resource)
		}
	}

	result.Coverage = agentdefinition.CoverageFull
	if len(result.MissingResources) > 0 {
		result.Coverage = agentdefinition.CoveragePartial
	}

	return result
}

func (e *engine) inheritedPermissions(
	role *permission.Role,
	byID map[pulid.ID]*permission.Role,
) *repositories.CachedPermissions {
	resources := make(map[string]*repositories.CachedResourcePermission)
	visited := make(map[pulid.ID]struct{}, len(byID))
	pending := []*permission.Role{role}

	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, seen := visited[current.ID]; seen {
			continue
		}
		visited[current.ID] = struct{}{}
		e.mergeRolePermissionsIntoCache(resources, current.Permissions)

		for _, parentID := range current.ParentRoleIDs {
			if parent, ok := byID[parentID]; ok {
				pending = append(pending, parent)
			}
		}
	}

	return &repositories.CachedPermissions{Resources: resources}
}
