package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
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

// ListLiveAssistantTurnsRequest asks for every reply one person still has in
// progress, across all of their conversations in a tenant.
type ListLiveAssistantTurnsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// LiveAssistantTurn is a turn still producing its reply, with the title of
// the conversation it is producing it in, so a list of them can be read
// without a second query per conversation.
type LiveAssistantTurn struct {
	conversation.AssistantTurn `bun:",extend"`

	ThreadTitle string `bun:"thread_title,scanonly"`
}

// CompleteAssistantTurnRequest closes a turn.
type CompleteAssistantTurnRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     conversation.AssistantTurnStatus
	Error      string
	RunID      pulid.ID
}

type RecordAssistantTurnFingerprintRequest struct {
	ID          pulid.ID
	TenantInfo  pagination.TenantInfo
	Fingerprint *agent.Fingerprint
}

type AssistantTurnRepository interface {
	RecordFingerprint(ctx context.Context, req RecordAssistantTurnFingerprintRequest) error
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
	// ListLive is every turn one person still has in progress, oldest first.
	// It is how a reply started in one tab is found from another, and what
	// signing out stops.
	ListLive(ctx context.Context, req ListLiveAssistantTurnsRequest) ([]*LiveAssistantTurn, error)
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
