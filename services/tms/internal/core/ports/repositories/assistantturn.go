package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrTurnAlreadyRunning reports a conversation that is already producing a
// reply. It is not a fault: it is the partial unique index doing its job, and
// the caller decides whether to stop the running turn or refuse the new one.
var ErrTurnAlreadyRunning = errors.New("this conversation already has a turn in progress")

type GetAssistantTurnRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	// UserID scopes the read to whoever asked the question. A turn belongs to
	// one person, and the relay hands out replies, so this is not optional
	// wherever a reader supplied the id.
	UserID pulid.ID
}

type ActiveAssistantTurnRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// CompleteAssistantTurnRequest closes a turn.
type CompleteAssistantTurnRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     conversation.AssistantTurnStatus
	Error      string
	RunID      pulid.ID
}

type AssistantTurnRepository interface {
	// Start records a turn about to run. A conversation that already has one
	// returns ErrTurnAlreadyRunning.
	Start(
		ctx context.Context,
		turn *conversation.AssistantTurn,
	) (*conversation.AssistantTurn, error)
	GetByID(ctx context.Context, req GetAssistantTurnRequest) (*conversation.AssistantTurn, error)
	// Active is the turn a conversation is still producing, nil when it is
	// not producing one. This is what lets a reopened tab rejoin a reply.
	Active(ctx context.Context, req ActiveAssistantTurnRequest) (*conversation.AssistantTurn, error)
	Complete(ctx context.Context, req CompleteAssistantTurnRequest) error
	// MarkWorkflow records the durable execution carrying the turn, so it can
	// be cancelled when somebody presses stop.
	MarkWorkflow(
		ctx context.Context,
		id pulid.ID,
		tenant pagination.TenantInfo,
		workflowID string,
	) error
}
