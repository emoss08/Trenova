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

type ListThreadsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
	Offset     int
}

type ListMessagesRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	// Limit bounds how much history is replayed to the model. Zero means all.
	Limit int
}

// AppendTurnRequest saves one exchange. The messages are written together and
// numbered together, because a half-saved turn would leave a tool call with no
// result and break the next replay.
type AppendTurnRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Messages   []conversation.Message
}

type ConversationRepository interface {
	CreateThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	GetThread(ctx context.Context, req GetThreadRequest) (*conversation.Thread, error)
	ListThreads(
		ctx context.Context,
		req ListThreadsRequest,
	) (*pagination.ListResult[*conversation.Thread], error)
	UpdateThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	DeleteThread(ctx context.Context, req GetThreadRequest) error
	ListMessages(ctx context.Context, req ListMessagesRequest) ([]conversation.Message, error)
	// AppendTurn allocates sequence numbers and writes the messages atomically,
	// returning them with their assigned identifiers.
	AppendTurn(ctx context.Context, req AppendTurnRequest) ([]conversation.Message, error)
}
