package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentDefinitionByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetAgentDefinitionBySystemKeyRequest struct {
	SystemKey  string
	TenantInfo pagination.TenantInfo
}

type ListAgentDefinitionRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	EnabledOnly bool
	ChatOnly    bool
	Audience    *AgentAudience
}

type ListAgentDefinitionConnectionRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	Cursor      pagination.CursorInfo    `json:"-"`
	Columns     []string                 `json:"-"`
	EnabledOnly bool                     `json:"-"`
	ChatOnly    bool                     `json:"-"`
	Audience    *AgentAudience           `json:"-"`
}

type ListAgentDefinitionsByTriggerRequest struct {
	TenantInfo pagination.TenantInfo
	Mode       agentdefinition.TriggerMode
	EventKind  agent.EventKind
}

type ListDueAgentDefinitionsRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	Limit      int
}

// ListScheduledAcrossTenantsRequest pages through every scheduled or continuous
// agent in every tenant, enabled or not, in id order.
type ListScheduledAcrossTenantsRequest struct {
	AfterID pulid.ID
	Limit   int
}

type MarkAgentDefinitionRunRequest struct {
	ID                pulid.ID
	TenantInfo        pagination.TenantInfo
	LastRunAt         int64
	NextRunAt         *int64
	ExpectedNextRunAt *int64
}

// SetAgentDefinitionToolTierRequest changes one tool's tier on an agent
// without rewriting the rest of the definition, so it cannot lose a change a
// person is saving at the same moment.
type SetAgentDefinitionToolTierRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	ToolName   string
	Tier       agent.AutonomyTier
}

type SetAgentDefinitionAccessModeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Mode       agentdefinition.AccessMode
}

type DeleteAgentDefinitionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentDefinitionStats struct {
	DefinitionID     pulid.ID `bun:"definition_id"`
	PendingProposals int      `bun:"pending_proposals"`
	OpenRuns         int      `bun:"open_runs"`
	LastRunAt        *int64   `bun:"last_run_at"`
}

type ListAgentDefinitionsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentDefinitionRepository interface {
	ListByIDs(
		ctx context.Context,
		req ListAgentDefinitionsByIDsRequest,
	) ([]*agentdefinition.Definition, error)
	List(
		ctx context.Context,
		req *ListAgentDefinitionRequest,
	) (*pagination.ListResult[*agentdefinition.Definition], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentDefinitionConnectionRequest,
	) (*pagination.CursorListResult[*agentdefinition.Definition], error)
	GetByID(
		ctx context.Context,
		req GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
	GetBySystemKey(
		ctx context.Context,
		req GetAgentDefinitionBySystemKeyRequest,
	) (*agentdefinition.Definition, error)
	ListEnabledByTrigger(
		ctx context.Context,
		req ListAgentDefinitionsByTriggerRequest,
	) ([]*agentdefinition.Definition, error)
	ListDue(
		ctx context.Context,
		req ListDueAgentDefinitionsRequest,
	) ([]*agentdefinition.Definition, error)
	ListScheduledAcrossTenants(
		ctx context.Context,
		req ListScheduledAcrossTenantsRequest,
	) ([]*agentdefinition.Definition, error)
	Create(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	Update(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	MarkRun(ctx context.Context, req MarkAgentDefinitionRunRequest) (bool, error)
	SetToolTier(ctx context.Context, req SetAgentDefinitionToolTierRequest) error
	SetAccessMode(ctx context.Context, req SetAgentDefinitionAccessModeRequest) error
	Delete(ctx context.Context, req DeleteAgentDefinitionRequest) error
	StatsByIDs(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]AgentDefinitionStats, error)
}
