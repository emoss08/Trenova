package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// SetAgentAccessRequest says who may use an agent: everyone who may use the
// assistant, or only the roles named. The roles are kept whatever the mode,
// so an agent opened to everyone and restricted again keeps its audience.
type SetAgentAccessRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
	Mode       agentdefinition.AccessMode
	RoleIDs    []pulid.ID
}

// SetRoleAgentsRequest replaces the agents a role is granted.
type SetRoleAgentsRequest struct {
	TenantInfo pagination.TenantInfo
	RoleID     pulid.ID
	AgentIDs   []pulid.ID
}

type SuggestAgentAudienceRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
}

// AgentAccess is an agent and the roles granted it.
type AgentAccess struct {
	Agent *agentdefinition.Definition
	Roles []*permission.Role
}

// RoleAgents is a role and the agents granted it.
type RoleAgents struct {
	Role   *permission.Role
	Agents []*agentdefinition.Definition
}

// AgentAudienceRole is how much of an agent one role could use, and whether
// it is granted the agent now.
type AgentAudienceRole struct {
	Role             *permission.Role
	Coverage         agentdefinition.AudienceCoverage
	MissingResources []permission.Resource
	Granted          bool
}

// AgentAudienceSuggestion is who an agent could be given to: every role in
// the tenant with its coverage, and, while the agent is open to everyone,
// the tools it holds that reach sensitive data.
type AgentAudienceSuggestion struct {
	Agent          *agentdefinition.Definition
	Roles          []AgentAudienceRole
	SensitiveTools []string
}

// AgentAccessService is the one writer of who may use which agent, behind
// both the agent form and the role editor.
type AgentAccessService interface {
	SetAgentAccess(
		ctx context.Context,
		req *SetAgentAccessRequest,
		actor *RequestActor,
	) (*AgentAccess, error)
	SetRoleAgents(
		ctx context.Context,
		req *SetRoleAgentsRequest,
		actor *RequestActor,
	) (*RoleAgents, error)
	SuggestAudience(
		ctx context.Context,
		req *SuggestAgentAudienceRequest,
	) (*AgentAudienceSuggestion, error)
}
