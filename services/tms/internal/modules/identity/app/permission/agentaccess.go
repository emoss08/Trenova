package permission

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreach"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (e *engine) grantedAgentIDs(
	ctx context.Context,
	orgID pulid.ID,
	roles []*rbac.Role,
) ([]string, error) {
	if len(roles) == 0 {
		return []string{}, nil
	}

	roleIDs := make([]pulid.ID, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			roleIDs = append(roleIDs, role.ID)
		}
	}

	granted, err := e.agentGrants.ListGrantedAgentIDs(
		ctx,
		permissionports.ListGrantedAgentIDsRequest{
			OrganizationID: orgID,
			RoleIDs:        roleIDs,
		},
	)
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
	actor *permissionports.RequestActor,
	operation permission.Operation,
) (*permissionports.UsableAgents, error) {
	if actor == nil {
		return &permissionports.UsableAgents{}, nil
	}

	req := actor.PermissionCheck(permission.ResourceAssistant, operation)
	if actor.IsAPIKey() || actor.IsAgent() {
		result, err := e.Check(ctx, req)
		if err != nil {
			return nil, err
		}

		return &permissionports.UsableAgents{
			Assistant:  result.Allowed,
			PlanDenial: result.PlanDenial,
		}, nil
	}

	denial, offPlan, err := e.newPlanGate().Resource(ctx, req.Resource)
	if err != nil {
		return nil, err
	}
	if offPlan {
		return &permissionports.UsableAgents{PlanDenial: &denial}, nil
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

	usable := &permissionports.UsableAgents{Assistant: result.Allowed}
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
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) (bool, error) {
	if actor == nil || definition == nil {
		return false, nil
	}

	usable, err := e.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		return false, err
	}
	if usable.PlanDenial != nil {
		return false, usable.PlanDenial.Restriction()
	}

	return usable.Allows(definition), nil
}

func (e *engine) RoleCoverage(
	ctx context.Context,
	req *permissionports.RoleCoverageRequest,
) ([]permissionports.RoleCoverage, error) {
	if req == nil || len(req.RoleIDs) == 0 {
		return []permissionports.RoleCoverage{}, nil
	}

	offPlan, err := e.newPlanGate().offPlan(ctx, coverageResources(req.Required)...)
	if err != nil {
		return nil, err
	}

	closure, err := e.roleRepo.GetRolesWithInheritance(ctx, req.RoleIDs)
	if err != nil {
		return nil, fmt.Errorf("read the roles and the roles they inherit: %w", err)
	}

	byID := make(map[pulid.ID]*rbac.Role, len(closure))
	for _, role := range closure {
		if role != nil && role.OrganizationID == req.OrganizationID {
			byID[role.ID] = role
		}
	}

	coverage := make([]permissionports.RoleCoverage, 0, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		role, ok := byID[roleID]
		if !ok {
			continue
		}
		coverage = append(coverage, e.coverageOf(role, byID, req.Required, offPlan))
	}

	return coverage, nil
}

func coverageResources(required []permissionports.RequiredGrant) []permission.Resource {
	resources := make([]permission.Resource, 0, len(required)+1)
	resources = append(resources, permission.ResourceAssistant)
	for _, grant := range required {
		resources = append(resources, grant.Resource)
	}

	return resources
}

func (e *engine) coverageOf(
	role *rbac.Role,
	byID map[pulid.ID]*rbac.Role,
	required []permissionports.RequiredGrant,
	offPlan map[permission.Resource]bool,
) permissionports.RoleCoverage {
	perms := e.inheritedPermissions(role, byID)
	result := permissionports.RoleCoverage{
		RoleID:           role.ID,
		Coverage:         agentreach.CoverageNone,
		MissingResources: []permission.Resource{},
	}
	if !e.grantedOnPlan(perms, offPlan, permission.ResourceAssistant, permission.OpCreate) {
		return result
	}

	for _, grant := range required {
		if e.grantedOnPlan(perms, offPlan, grant.Resource, grant.Operation) {
			continue
		}
		if !slices.Contains(result.MissingResources, grant.Resource) {
			result.MissingResources = append(result.MissingResources, grant.Resource)
		}
	}

	result.Coverage = agentreach.CoverageFull
	if len(result.MissingResources) > 0 {
		result.Coverage = agentreach.CoveragePartial
	}

	return result
}

func (e *engine) grantedOnPlan(
	perms *permissionports.CachedPermissions,
	offPlan map[permission.Resource]bool,
	resource permission.Resource,
	operation permission.Operation,
) bool {
	if offPlan[resource] {
		return false
	}
	_, ok := e.resolveResourcePermission(perms, resource.String(), operation)

	return ok
}

func (e *engine) inheritedPermissions(
	role *rbac.Role,
	byID map[pulid.ID]*rbac.Role,
) *permissionports.CachedPermissions {
	resources := make(map[string]*permissionports.CachedResourcePermission)
	visited := make(map[pulid.ID]struct{}, len(byID))
	pending := []*rbac.Role{role}

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

	return &permissionports.CachedPermissions{Resources: resources}
}
