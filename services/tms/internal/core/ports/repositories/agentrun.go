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

type AgentRunRepository interface {
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
}
