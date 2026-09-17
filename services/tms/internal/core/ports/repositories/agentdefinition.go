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
	Filter *pagination.QueryOptions `json:"filter"`
	// EnabledOnly narrows to agents a person can actually start a conversation
	// with.
	EnabledOnly bool
	// ChatOnly narrows to agents that hold conversations, since a scheduled or
	// event-driven agent has no thread to talk in.
	ChatOnly bool
}

type ListAgentDefinitionConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type ListAgentDefinitionsByTriggerRequest struct {
	TenantInfo pagination.TenantInfo
	Mode       agentdefinition.TriggerMode
	// EventKind narrows event-triggered definitions to those subscribed to it.
	EventKind agent.EventKind
}

type ListDueAgentDefinitionsRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	Limit      int
}

type MarkAgentDefinitionRunRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	LastRunAt  int64
	// NextRunAt is the slot the sweep claimed; nil clears it, which is how a
	// definition that reached its end stops firing.
	NextRunAt *int64
	// ExpectedNextRunAt is the slot the caller saw. The update applies only when
	// the row still carries it, so two sweeps cannot both claim one slot.
	ExpectedNextRunAt *int64
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

type AgentDefinitionRepository interface {
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
	Create(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	Update(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	MarkRun(ctx context.Context, req MarkAgentDefinitionRunRequest) (bool, error)
	Delete(ctx context.Context, req DeleteAgentDefinitionRequest) error
	StatsByIDs(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]AgentDefinitionStats, error)
}
