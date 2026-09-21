package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentProposalRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	ExcludeShadowDefinitions bool                     `json:"-"`
}

type ListAgentProposalConnectionRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	Cursor                   pagination.CursorInfo    `json:"-"`
	Columns                  []string                 `json:"-"`
	ExcludeShadowDefinitions bool                     `json:"-"`
}

type GetAgentProposalByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type UpdateAgentProposalStatusRequest struct {
	ID     pulid.ID             `json:"id"`
	Status agent.ProposalStatus `json:"status"`
	// FromStatus, when set, makes the update conditional: the row changes only
	// if it still holds this status, and a race that lost is reported as a
	// conflict rather than silently overwriting the winner.
	FromStatus agent.ProposalStatus  `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// RecordAgentProposalExecutionRequest records what happened when an approved
// proposal's tool ran, so "approved" and "approved and done" stay distinguishable.
type RecordAgentProposalExecutionRequest struct {
	ID             pulid.ID              `json:"id"`
	Status         agent.ProposalStatus  `json:"status"`
	ExecutedAt     *int64                `json:"executedAt"`
	ExecutionError string                `json:"executionError"`
	TenantInfo     pagination.TenantInfo `json:"-"`
}

// ListAgentProposalsByThreadRequest fetches every proposal raised during one
// assistant conversation, so reopening a thread shows what is still waiting on a
// decision rather than only what the last turn returned.
type ListAgentProposalsByThreadRequest struct {
	ThreadID   pulid.ID              `json:"threadId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ExpireAgentProposalsByRunRequest struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// ExpireAgentProposalsRequest expires every pending proposal, in any tenant,
// whose expiry has passed. It is deliberately unscoped: it is the sweeper's
// request, and the sweeper runs for the whole system.
type ExpireAgentProposalsRequest struct {
	Before int64 `json:"before"`
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
	ListByThread(
		ctx context.Context,
		req ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
	UpdateStatus(
		ctx context.Context,
		req UpdateAgentProposalStatusRequest,
	) (*agent.AgentProposal, error)
	ExpirePendingByRun(ctx context.Context, req ExpireAgentProposalsByRunRequest) (int, error)
	ExpirePending(ctx context.Context, req ExpireAgentProposalsRequest) (int, error)
	RecordExecution(
		ctx context.Context,
		req RecordAgentProposalExecutionRequest,
	) (*agent.AgentProposal, error)
}
