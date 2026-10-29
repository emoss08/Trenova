package agentdecisionqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PendingDecision is one thing waiting on a person: a proposal that stands
// on its own, or a plan decided as a whole. Exactly one of the two is set.
type PendingDecision struct {
	Proposal  *agent.AgentProposal
	Plan      *agentplan.AgentPlan
	CreatedAt int64
	Hold      *assistantproposalports.ProposalHold
	// Cursor is where a page continues from after this entry.
	Cursor string
}

type ListPendingDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	First      int
	After      string
	// AgentDefinitionID narrows the queue to one agent's work; ToolName to
	// one kind of write (a plan matches when any step uses it).
	AgentDefinitionID pulid.ID
	ToolName          string
	IncludeTotalCount bool
	// Usable keeps only what agents the reader may use raised. Nil keeps
	// everything, for a caller reading the queue for the organization.
	Usable *permissionports.UsableAgents
}

type ListRecentDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	// Since is the earliest decision kept, in Unix seconds.
	Since int64
	First int
	// Usable keeps only what agents the reader may use raised. Nil keeps
	// everything.
	Usable *permissionports.UsableAgents
}

// RecentDecision is one decision on a proposal, with the proposal as it
// stands now and the decider's name.
type RecentDecision struct {
	RecentDecisionEntry
	Proposal *agent.AgentProposal
}

type PendingDecisionsPage struct {
	Items           []PendingDecision
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
}

// DecideAgentProposalsRequest decides several proposals of the same tool
// at once, each as proposed. A change to one proposal is a decision on
// that proposal alone.
type DecideAgentProposalsRequest struct {
	ProposalIDs []pulid.ID
	Decision    agentdecision.DecisionType
	ReasonCode  string
	Note        string
	TenantInfo  pagination.TenantInfo
	// PreviewDigests is the digest of the preview the decider was shown for
	// each proposal. One that no longer matches fails that proposal alone; a
	// proposal without one is recorded as approved unreviewed.
	PreviewDigests map[pulid.ID]string
}

type AgentDecisionQueueService interface {
	ListPending(ctx context.Context, req ListPendingDecisionsRequest) (*PendingDecisionsPage, error)
	// Count is the size of the queue, for a badge. Usable narrows it to what
	// the reader's agents raised; nil counts everything.
	Count(
		ctx context.Context,
		tenant pagination.TenantInfo,
		usable *permissionports.UsableAgents,
	) (int, error)
	Summary(
		ctx context.Context,
		tenant pagination.TenantInfo,
		usable *permissionports.UsableAgents,
	) (*PendingDecisionSummary, error)
	DecideMany(
		ctx context.Context,
		req *DecideAgentProposalsRequest,
		actor *permissionports.RequestActor,
	) ([]agentdecisionports.AgentProposalDecisionResult, error)
	// Recent is what was decided since a moment, newest first, each with
	// the proposal it decided and who decided it.
	Recent(ctx context.Context, req ListRecentDecisionsRequest) ([]RecentDecision, error)
	DecideManyOwn(
		ctx context.Context,
		req *DecideAgentProposalsRequest,
		actor *permissionports.RequestActor,
	) ([]agentdecisionports.AgentProposalDecisionResult, error)
}
