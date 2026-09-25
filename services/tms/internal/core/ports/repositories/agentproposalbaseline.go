package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetProposalBaselineRequest struct {
	ProposalID pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListProposalBaselinesRequest struct {
	ProposalIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
}

// PurgeOrphanProposalBaselinesRequest removes baselines taken before Before
// whose proposal was never filed, at most Limit at a time, in every tenant.
type PurgeOrphanProposalBaselinesRequest struct {
	Before int64
	Limit  int
}

// AgentProposalBaselineRepository keeps what a write would have done when it
// was proposed, beside the proposal rather than on it.
type AgentProposalBaselineRepository interface {
	// Create keeps a baseline; one already kept for the proposal stays.
	Create(ctx context.Context, baseline *agent.ProposalBaseline) error
	GetByProposal(
		ctx context.Context,
		req GetProposalBaselineRequest,
	) (*agent.ProposalBaseline, error)
	ListByProposals(
		ctx context.Context,
		req ListProposalBaselinesRequest,
	) ([]*agent.ProposalBaseline, error)
	PurgeOrphans(ctx context.Context, req PurgeOrphanProposalBaselinesRequest) (int, error)
}
