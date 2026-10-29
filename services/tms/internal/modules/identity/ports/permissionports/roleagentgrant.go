package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListGrantedAgentIDsRequest struct {
	OrganizationID pulid.ID
	RoleIDs        []pulid.ID
}

type ListGrantsByAgentsRequest struct {
	TenantInfo pagination.TenantInfo
	AgentIDs   []pulid.ID
}

type ListGrantsByRolesRequest struct {
	TenantInfo pagination.TenantInfo
	RoleIDs    []pulid.ID
}

// ListGrantableRolesRequest reads the tenant's roles that may be granted an
// agent: the ones named, or every one when none is.
type ListGrantableRolesRequest struct {
	TenantInfo pagination.TenantInfo
	RoleIDs    []pulid.ID
}

type ReplaceAgentGrantsRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
	RoleIDs    []pulid.ID
	GrantedBy  pulid.ID
}

type ReplaceRoleGrantsRequest struct {
	TenantInfo pagination.TenantInfo
	RoleID     pulid.ID
	AgentIDs   []pulid.ID
	GrantedBy  pulid.ID
}

// GrantChange is what a replace did: the ids it granted and took away, and
// what was granted before it ran.
type GrantChange struct {
	Previous []pulid.ID
	Added    []pulid.ID
	Removed  []pulid.ID
}

func (c GrantChange) Changed() bool {
	return len(c.Added) > 0 || len(c.Removed) > 0
}

// RoleAgentGrantRepository keeps which roles may use which agents restricted
// to roles. Its replaces run inside the caller's transaction when there is
// one.
type RoleAgentGrantRepository interface {
	ListGrantedAgentIDs(ctx context.Context, req ListGrantedAgentIDsRequest) ([]pulid.ID, error)
	ListRolesByAgents(
		ctx context.Context,
		req ListGrantsByAgentsRequest,
	) (map[pulid.ID][]*rbac.Role, error)
	ListAgentsByRoles(
		ctx context.Context,
		req ListGrantsByRolesRequest,
	) (map[pulid.ID][]*agentdefinition.Definition, error)
	ListGrantableRoles(
		ctx context.Context,
		req ListGrantableRolesRequest,
	) ([]*rbac.Role, error)
	ReplaceForAgent(ctx context.Context, req ReplaceAgentGrantsRequest) (GrantChange, error)
	ReplaceForRole(ctx context.Context, req ReplaceRoleGrantsRequest) (GrantChange, error)
}
