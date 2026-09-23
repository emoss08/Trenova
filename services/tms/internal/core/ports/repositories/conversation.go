package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetThreadRequest reads one thread. UserID is required rather than optional:
// threads are per-person, so every read is scoped to the person reading.
type GetThreadRequest struct {
	ID         pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetThreadOwnedRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListThreadsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
	Offset     int
	// IncludeUnlisted also returns conversations whose origin keeps them out
	// of the rail, such as quick questions that were not kept.
	IncludeUnlisted bool
}

type ListMessagesRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	// Limit bounds how much history is replayed to the model. Zero means all.
	Limit int
	// BeforeSequence, when set, reads only messages numbered below it: the
	// page above the one a reader already has. Sequence is the thread's own
	// order, so a page cut here never repeats or skips a message however
	// many turns land while the reader scrolls.
	BeforeSequence *int
	// ExcludeKinds leaves messages of these kinds out, before Limit counts:
	// the history replayed to the model leaves out another agent's steps, so
	// they neither reach the model nor crowd its window.
	ExcludeKinds []conversation.MessageKind
}

// CountMessagesRequest counts a thread's messages, which is how long the
// conversation is.
type CountMessagesRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

// AppendTurnRequest saves one exchange. The messages are written together and
// numbered together, because a half-saved turn would leave a tool call with no
// result and break the next replay.
type AppendTurnRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Messages   []conversation.Message
}

// DeleteStaleThreadsRequest names the unkept conversations to remove: those
// of one origin whose last activity is older than Before, at most Limit at a
// time so a sweep never holds a long transaction.
type DeleteStaleThreadsRequest struct {
	Origin conversation.ThreadOrigin
	Before int64
	Limit  int
}

type ConversationRepository interface {
	CreateThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	GetThread(ctx context.Context, req GetThreadRequest) (*conversation.Thread, error)
	// GetThreadOwned reads a thread within a tenant whoever owns it, so the
	// application can act in a conversation on its owner's behalf. It is
	// never reached from a request: a person reads threads through GetThread,
	// scoped to themselves.
	GetThreadOwned(ctx context.Context, req GetThreadOwnedRequest) (*conversation.Thread, error)
	ListThreads(
		ctx context.Context,
		req ListThreadsRequest,
	) (*pagination.ListResult[*conversation.Thread], error)
	UpdateThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	DeleteThread(ctx context.Context, req GetThreadRequest) error
	// DeleteStaleThreads removes unkept conversations of an origin, with their
	// messages, across every tenant. It reports how many threads went.
	DeleteStaleThreads(ctx context.Context, req DeleteStaleThreadsRequest) (int, error)
	ListMessages(ctx context.Context, req ListMessagesRequest) ([]conversation.Message, error)
	CountMessages(ctx context.Context, req CountMessagesRequest) (int, error)
	// AppendTurn allocates sequence numbers and writes the messages atomically,
	// returning them with their assigned identifiers.
	AppendTurn(ctx context.Context, req AppendTurnRequest) ([]conversation.Message, error)
}
