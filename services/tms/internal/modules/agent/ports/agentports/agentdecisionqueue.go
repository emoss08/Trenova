package agentports

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PendingDecisionKind says which record a queue entry stands for. A plan is
// one decision however many steps it has; its steps never appear on their
// own.
type PendingDecisionKind string

const PendingDecisionProposal PendingDecisionKind = "proposal"

const PendingDecisionPlan PendingDecisionKind = "plan"

// PendingDecisionEntry is one thing waiting on a person, newest first.
type PendingDecisionEntry struct {
	Kind      PendingDecisionKind
	ID        pulid.ID
	CreatedAt int64
	Shadow    bool
}

// PendingDecisionCursor is where a page ended: the entry's time and id,
// which together order the queue.
type PendingDecisionCursor struct {
	CreatedAt int64
	ID        pulid.ID
}

type ListPendingDecisionsQuery struct {
	TenantInfo pagination.TenantInfo
	// Limit is the page size; the repository reads one more to learn
	// whether a next page exists.
	Limit int
	After *PendingDecisionCursor
	// AgentDefinitionID narrows the queue to one agent's work; ToolName to
	// one kind of write.
	AgentDefinitionID pulid.ID
	ToolName          string
	// Audience keeps only what agents the person may use raised; nil keeps
	// everything.
	Audience *kernelports.AgentAudience
	// Now is the clock the expiry check reads.
	Now            int64
	ActionableOnly bool
}

type PendingDecisionsPageRow struct {
	Entries     []PendingDecisionEntry
	HasNextPage bool
}

type PendingDecisionAgentCount struct {
	AgentDefinitionID pulid.ID
	AgentName         string
	Count             int
}

type PendingDecisionToolCount struct {
	ToolName string
	Count    int
}

// PendingDecisionSummary is the queue in numbers: how much is waiting, from
// which agents, of which kinds, and how long the oldest has waited.
type PendingDecisionSummary struct {
	Total    int
	ByAgent  []PendingDecisionAgentCount
	ByTool   []PendingDecisionToolCount
	OldestAt *int64
}

type PendingDecisionSummaryRequest struct {
	TenantInfo pagination.TenantInfo
	Audience   *kernelports.AgentAudience
	Now        int64
}

// AgentDecisionQueueRepository reads what is waiting on a person across
// proposals and plans as one queue. It answers with entries, not records:
// the caller loads each record from its own repository, so the queue never
// becomes a second copy of either.
type AgentDecisionQueueRepository interface {
	ListPending(
		ctx context.Context,
		req ListPendingDecisionsQuery,
	) (*PendingDecisionsPageRow, error)
	// CountPending is the size of the queue under the same filters as
	// ListPending, for a connection that asks for its total.
	CountPending(ctx context.Context, req ListPendingDecisionsQuery) (keel.RowCount, error)
	Summary(ctx context.Context, req PendingDecisionSummaryRequest) (*PendingDecisionSummary, error)
	// ListRecent is what was decided lately, newest first: proposals that
	// stood on their own, each with who decided it.
	ListRecent(ctx context.Context, req ListRecentDecisionsQuery) ([]RecentDecisionEntry, error)
}

type ListRecentDecisionsQuery struct {
	TenantInfo pagination.TenantInfo
	// Since is the earliest decision kept, in Unix seconds.
	Since int64
	Limit int
	// Audience keeps only what agents the person may use raised; nil keeps
	// everything.
	Audience *kernelports.AgentAudience
}

// RecentDecisionEntry is one decision on a proposal and the person who made
// it, by name as the app shows them.
type RecentDecisionEntry struct {
	DecisionID      pulid.ID
	ProposalID      pulid.ID
	Decision        string
	ReasonCode      string
	Note            string
	DecidedByUserID pulid.ID
	DecidedByName   string
	DecidedAt       int64
}
