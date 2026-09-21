package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PendingDecisionKind says which record a queue entry stands for. A plan is
// one decision however many steps it has; its steps never appear on their
// own.
type PendingDecisionKind string

const (
	PendingDecisionProposal PendingDecisionKind = "proposal"
	PendingDecisionPlan     PendingDecisionKind = "plan"
)

// PendingDecisionEntry is one thing waiting on a person, newest first.
type PendingDecisionEntry struct {
	Kind      PendingDecisionKind `bun:"kind"`
	ID        pulid.ID            `bun:"id"`
	CreatedAt int64               `bun:"created_at"`
}

// PendingDecisionCursor is where a page ended: the entry's time and id,
// which together order the queue.
type PendingDecisionCursor struct {
	CreatedAt int64
	ID        pulid.ID
}

type ListPendingDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	// Limit is the page size; the repository reads one more to learn
	// whether a next page exists.
	Limit int
	After *PendingDecisionCursor
	// AgentDefinitionID narrows the queue to one agent's work; ToolName to
	// one kind of write.
	AgentDefinitionID pulid.ID
	ToolName          string
	// ExcludeShadowDefinitions hides what agents in shadow mode proposed,
	// which nobody can approve.
	ExcludeShadowDefinitions bool
	// Now is the clock the expiry check reads.
	Now int64
}

type PendingDecisionsPage struct {
	Entries     []PendingDecisionEntry
	HasNextPage bool
}

type PendingDecisionAgentCount struct {
	AgentDefinitionID pulid.ID `bun:"agent_definition_id"`
	AgentName         string   `bun:"agent_name"`
	Count             int      `bun:"count"`
}

type PendingDecisionToolCount struct {
	ToolName string `bun:"tool_name"`
	Count    int    `bun:"count"`
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
	TenantInfo               pagination.TenantInfo
	ExcludeShadowDefinitions bool
	Now                      int64
}

// AgentDecisionQueueRepository reads what is waiting on a person across
// proposals and plans as one queue. It answers with entries, not records:
// the caller loads each record from its own repository, so the queue never
// becomes a second copy of either.
type AgentDecisionQueueRepository interface {
	ListPending(ctx context.Context, req ListPendingDecisionsRequest) (*PendingDecisionsPage, error)
	// CountPending is the size of the queue under the same filters as
	// ListPending, for a connection that asks for its total.
	CountPending(ctx context.Context, req ListPendingDecisionsRequest) (int, error)
	Summary(ctx context.Context, req PendingDecisionSummaryRequest) (*PendingDecisionSummary, error)
}
