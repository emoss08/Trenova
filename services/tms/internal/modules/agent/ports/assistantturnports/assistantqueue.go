package assistantturnports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrQueueFull = errors.New("the conversation's queue is full")

var ErrQueueEmpty = errors.New("no queued message to take")

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
	List(
		ctx context.Context,
		scope *AssistantQueueScope,
	) ([]*conversationqueue.QueuedMessage, error)
	Get(ctx context.Context, req *QueuedMessageRequest) (*conversationqueue.QueuedMessage, error)
	Insert(
		ctx context.Context,
		entity *conversationqueue.QueuedMessage,
	) (*conversationqueue.QueuedMessage, error)
	Update(
		ctx context.Context,
		entity *conversationqueue.QueuedMessage,
	) (*conversationqueue.QueuedMessage, error)
	MarkSteer(
		ctx context.Context,
		req *QueuedMessageRequest,
	) (*conversationqueue.QueuedMessage, error)
	Delete(ctx context.Context, req *QueuedMessageRequest) error
	DeleteMany(ctx context.Context, req *DeleteQueuedMessagesRequest) error
	Reorder(
		ctx context.Context,
		req *ReorderQueuedMessagesRequest,
	) ([]*conversationqueue.QueuedMessage, error)
	ClaimNext(
		ctx context.Context,
		scope *AssistantQueueScope,
	) (*conversationqueue.QueuedMessage, error)
	Claim(ctx context.Context, req *QueuedMessageRequest) (*conversationqueue.QueuedMessage, error)
	Restore(ctx context.Context, entity *conversationqueue.QueuedMessage) error
}
