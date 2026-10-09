package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrQueueFull  = errors.New("the conversation's queue is full")
	ErrQueueEmpty = errors.New("no queued message to take")
)

type AssistantQueueScope struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type QueuedMessageRequest struct {
	ID    pulid.ID
	Scope AssistantQueueScope
}

type DeleteQueuedMessagesRequest struct {
	IDs   []pulid.ID
	Scope AssistantQueueScope
}

type ReorderQueuedMessagesRequest struct {
	IDs   []pulid.ID
	Scope AssistantQueueScope
}

type AssistantQueueRepository interface {
	List(ctx context.Context, scope *AssistantQueueScope) ([]*conversation.QueuedMessage, error)
	Get(ctx context.Context, req *QueuedMessageRequest) (*conversation.QueuedMessage, error)
	Insert(
		ctx context.Context,
		entity *conversation.QueuedMessage,
	) (*conversation.QueuedMessage, error)
	Update(
		ctx context.Context,
		entity *conversation.QueuedMessage,
	) (*conversation.QueuedMessage, error)
	MarkSteer(ctx context.Context, req *QueuedMessageRequest) (*conversation.QueuedMessage, error)
	Delete(ctx context.Context, req *QueuedMessageRequest) error
	DeleteMany(ctx context.Context, req *DeleteQueuedMessagesRequest) error
	Reorder(
		ctx context.Context,
		req *ReorderQueuedMessagesRequest,
	) ([]*conversation.QueuedMessage, error)
	ClaimNext(ctx context.Context, scope *AssistantQueueScope) (*conversation.QueuedMessage, error)
	Claim(ctx context.Context, req *QueuedMessageRequest) (*conversation.QueuedMessage, error)
	Restore(ctx context.Context, entity *conversation.QueuedMessage) error
}
