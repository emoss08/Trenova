package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AppendAgentRunEventsRequest carries one owner's events, in order.
//
// Events arrive in batches rather than one at a time because a run emits them
// far faster than a round trip to postgres: writing each one where it happens
// would make recording the work slower than doing it.
type AppendAgentRunEventsRequest struct {
	TenantInfo pagination.TenantInfo
	Events     []*agent.AgentRunEvent
}

// ListAgentRunEventsRequest reads one owner's account of itself, in sequence.
type ListAgentRunEventsRequest struct {
	TenantInfo pagination.TenantInfo
	OwnerKind  string
	OwnerID    pulid.ID
	// After reads only what follows a sequence already seen, so a caller
	// following a run in progress does not re-read what it has.
	After int
	Limit int
}

// ListAgentRunEventConnectionRequest is the paged read behind GraphQL.
type ListAgentRunEventConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
	// OwnerKind keeps the list to one kind of owner. An assistant turn is one
	// person's conversation, and its events are theirs; a list read by
	// anyone who may read agent runs must not include them.
	OwnerKind string `json:"-"`
}

// PruneAgentRunEventsRequest drops events older than a cutoff.
type PruneAgentRunEventsRequest struct {
	Before int64
	Limit  int
}

// AgentRunEventRepository stores what a run did.
//
// There is no update and no upsert, and that absence is the point: an event is
// a record of something that already happened, and a log whose rows can be
// revised is not a log.
type AgentRunEventRepository interface {
	Append(ctx context.Context, req AppendAgentRunEventsRequest) error
	NextSequence(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ownerKind string,
		ownerID pulid.ID,
	) (int, error)
	List(ctx context.Context, req ListAgentRunEventsRequest) ([]*agent.AgentRunEvent, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentRunEventConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRunEvent], error)
	Prune(ctx context.Context, req PruneAgentRunEventsRequest) (int, error)
}
