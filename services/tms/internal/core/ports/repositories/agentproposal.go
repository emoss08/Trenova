package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentProposalRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListAgentProposalConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type GetAgentProposalByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type UpdateAgentProposalStatusRequest struct {
	ID         pulid.ID              `json:"id"`
	Status     agent.ProposalStatus  `json:"status"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ExpireAgentProposalsByRunRequest struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type AgentProposalRepository interface {
	List(
		ctx context.Context,
		req *ListAgentProposalRequest,
	) (*pagination.ListResult[*agent.AgentProposal], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentProposalConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentProposal], error)
	GetByID(ctx context.Context, req GetAgentProposalByIDRequest) (*agent.AgentProposal, error)
	Create(ctx context.Context, entity *agent.AgentProposal) (*agent.AgentProposal, error)
	UpdateStatus(
		ctx context.Context,
		req UpdateAgentProposalStatusRequest,
	) (*agent.AgentProposal, error)
	ExpirePendingByRun(ctx context.Context, req ExpireAgentProposalsByRunRequest) (int, error)
}
