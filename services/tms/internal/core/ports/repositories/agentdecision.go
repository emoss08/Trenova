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
type ListAgentDecisionsByProposalsRequest struct {
	ProposalIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
}

type AgentDecisionRepository interface {
	GetByID(ctx context.Context, req GetAgentDecisionByIDRequest) (*agent.AgentDecision, error)
	Create(ctx context.Context, entity *agent.AgentDecision) (*agent.AgentDecision, error)
	ListByProposals(
		ctx context.Context,
		req ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
}
