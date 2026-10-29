package agentproposalports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentProposalService interface {
	List(
		ctx context.Context,
		req *ListAgentProposalRequest,
	) (*pagination.ListResult[*agent.AgentProposal], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentProposalConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentProposal], error)
	GetByID(
		ctx context.Context,
		req GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

// PendingProposalsNotice is what the recorder hands the notifier once a run's
// proposals are stored: the agent, the run and the proposals themselves.
type PendingProposalsNotice struct {
	Definition *agentdefinition.Definition
	Run        *agent.AgentRun
	Proposals  []*agent.AgentProposal
}

// RemindPendingProposalsRequest asks for every proposal from a background run
// that has waited longer than OlderThan to be brought to its deciders again.
type RemindPendingProposalsRequest struct {
	Now       int64
	OlderThan time.Duration
	Limit     int
}

// AgentProposalNotifier tells the people who can decide a proposal that one
// is waiting. A proposal a person cannot see from where they are is a decision
// that never gets made; the panel is not where a dispatcher lives.
type AgentProposalNotifier interface {
	NotifyPending(ctx context.Context, notice PendingProposalsNotice) error
	RemindPending(ctx context.Context, req RemindPendingProposalsRequest) (int, error)
}
