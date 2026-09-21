package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentRunRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListAgentRunConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type GetAgentRunByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type CountOpenAgentRunsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	SubjectID    pulid.ID
}

// CountAgentRunsSinceRequest counts an agent's runs started at or after an
// instant, whatever became of them; a run that failed still spent its start.
type CountAgentRunsSinceRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	Since        int64
}

// ListAgentRunsByIDsRequest reads several runs at once, for a loader that
// resolves the run behind a page of proposals or plans.
type ListAgentRunsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentRunRepository interface {
	ListByIDs(ctx context.Context, req ListAgentRunsByIDsRequest) ([]*agent.AgentRun, error)
	List(
		ctx context.Context,
		req *ListAgentRunRequest,
	) (*pagination.ListResult[*agent.AgentRun], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentRunConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRun], error)
	GetByID(ctx context.Context, req GetAgentRunByIDRequest) (*agent.AgentRun, error)
	Create(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
	Update(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
	CountOpen(ctx context.Context, req CountOpenAgentRunsRequest) (int, error)
	CountSince(ctx context.Context, req CountAgentRunsSinceRequest) (int, error)
}
