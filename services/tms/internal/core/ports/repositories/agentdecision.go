package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentDecisionByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

// ListAgentDecisionsByProposalsRequest reads the decisions made on a set
// of proposals, newest first, so the latest decision on each is found first.
// A decision undone in its undo window is not one and is left out.
type ListAgentDecisionsByProposalsRequest struct {
	ProposalIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
}

// ListAgentDecisionsByCommitWorkflowRequest reads every decision one undo
// window holds: one for a proposal approved on its own, several for a batch.
type ListAgentDecisionsByCommitWorkflowRequest struct {
	WorkflowID string
	TenantInfo pagination.TenantInfo
}

// SettleAgentDecisionsRequest closes the undo window on decisions that are
// still in it: as committed, or as undone by UndoneByUserID. A decision
// already committed or undone is left as it is, so a commit and an undo
// racing each other settle every decision one way or the other.
type SettleAgentDecisionsRequest struct {
	IDs            []pulid.ID
	TenantInfo     pagination.TenantInfo
	At             int64
	UndoneByUserID pulid.ID
}

type AgentDecisionRepository interface {
	GetByID(ctx context.Context, req GetAgentDecisionByIDRequest) (*agent.AgentDecision, error)
	Create(ctx context.Context, entity *agent.AgentDecision) (*agent.AgentDecision, error)
	ListByProposals(
		ctx context.Context,
		req ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
	ListByCommitWorkflow(
		ctx context.Context,
		req ListAgentDecisionsByCommitWorkflowRequest,
	) ([]*agent.AgentDecision, error)
	// MarkCommitted and MarkUndone return the decisions they settled.
	MarkCommitted(ctx context.Context, req SettleAgentDecisionsRequest) ([]pulid.ID, error)
	MarkUndone(ctx context.Context, req SettleAgentDecisionsRequest) ([]pulid.ID, error)
}
