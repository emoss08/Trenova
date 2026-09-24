package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AgentAudience narrows a read of agents, or of what they raised, to the
// agents a person may use: every agent open to everyone, and the restricted
// ones their roles grant.
type AgentAudience struct {
	GrantedAgentIDs []pulid.ID
}

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
	) (map[pulid.ID][]*permission.Role, error)
	ListAgentsByRoles(
		ctx context.Context,
		req ListGrantsByRolesRequest,
	) (map[pulid.ID][]*agentdefinition.Definition, error)
	ListGrantableRoles(
		ctx context.Context,
		req ListGrantableRolesRequest,
	) ([]*permission.Role, error)
	ReplaceForAgent(ctx context.Context, req ReplaceAgentGrantsRequest) (GrantChange, error)
	ReplaceForRole(ctx context.Context, req ReplaceRoleGrantsRequest) (GrantChange, error)
}
