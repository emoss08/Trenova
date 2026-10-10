package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
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

type ListThreadAgentsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListThreadsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
	Cursor     string
	Until      string
	// IncludeUnlisted also returns conversations whose origin keeps them out
	// of the rail, such as quick questions that were not kept.
	IncludeUnlisted bool
}

type ThreadPage struct {
	Items      []*conversation.Thread `json:"items"`
	NextCursor string                 `json:"nextCursor,omitempty"`
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
	AfterSequence  *int
	// ExcludeKinds leaves messages of these kinds out, before Limit counts:
	// the history replayed to the model leaves out another agent's steps, so
	// they neither reach the model nor crowd its window.
	ExcludeKinds []conversation.MessageKind
	// Kinds, when set, reads only messages of these kinds: the decision notes
	// a conversation already carries, without the turns around them.
	Kinds []conversation.MessageKind
	// SinceCompaction reads only what the model still reads once the
	// conversation has been compacted: the latest summary, and every message
	// after the stretch it stands in for. Limit then counts from there. A
	// conversation never compacted is read as it would be without it.
	SinceCompaction bool
}

type AddSavedMemoriesRequest struct {
	MessageID  pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Memories   []conversation.SavedMemory
}

// UpdateThreadContextRequest keeps how full a conversation's context is, and
// whether it still compacts itself. A nil field is left as it is.
type UpdateThreadContextRequest struct {
	ThreadID       pulid.ID
	TenantInfo     pagination.TenantInfo
	Usage          *conversation.ContextUsage
	AutoCompactOff *bool
	WorkingSet     *[]conversation.WorkingRecord
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
	Origin          conversation.ThreadOrigin
	Before          int64
	Limit           int
	SubjectlessOnly bool
}

// MarkThreadTaintedRequest keeps the outside content a conversation has
// read, so every later turn in it opens tainted.
type MarkThreadTaintedRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Taint      *agent.RunTaint
	TaintedAt  int64
}

type ListThreadAttentionRequest struct {
	ThreadIDs  []pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type ThreadAttentionRow struct {
	PendingPlans        int
	PendingProposalRuns []pulid.ID
	LastTurnStatus      conversation.AssistantTurnStatus
}

type WakeThreadRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type MarkThreadReadRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	ReadAt     int64
}

type SearchMentionsRequest struct {
	TenantInfo   pagination.TenantInfo
	Query        string
	Kinds        []string
	LimitPerKind int
	// Offset skips that many rows of each kind, for a list paging through
	// one kind.
	Offset int
}

type MentionRow struct {
	Type     string `bun:"type"`
	ID       string `bun:"id"`
	Label    string `bun:"label"`
	Subtitle string `bun:"subtitle"`
}

// SearchDeskRequest searches what one person has in the Desk: their
// conversations, what was said in them, and what they produced.
type SearchDeskRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Query      string
	// Kinds is any of chat, msg, art and dec.
	Kinds        []string
	LimitPerKind int
}

type CountQuestionsSinceRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Since      int64
}

type DeskSearchRow struct {
	Kind         string   `bun:"kind"`
	ID           string   `bun:"id"`
	ThreadID     pulid.ID `bun:"thread_id"`
	AgentID      pulid.ID `bun:"agent_id"`
	Title        string   `bun:"title"`
	ThreadTitle  string   `bun:"thread_title"`
	ArtifactKind string   `bun:"artifact_kind"`
	Status       string   `bun:"status"`
	At           int64    `bun:"at"`
}

type ConversationRepository interface {
	SearchMentions(ctx context.Context, req SearchMentionsRequest) ([]MentionRow, error)
	SearchDesk(ctx context.Context, req SearchDeskRequest) ([]DeskSearchRow, error)
	// CountQuestionsSince counts what one person asked the agents from since
	// on, across all their conversations.
	CountQuestionsSince(ctx context.Context, req CountQuestionsSinceRequest) (int, error)
	ListThreadAttention(
		ctx context.Context,
		req ListThreadAttentionRequest,
	) (map[pulid.ID]ThreadAttentionRow, error)
	MarkThreadRead(ctx context.Context, req MarkThreadReadRequest) error
	// WakeThread ends a snooze the conversation is under, reporting whether
	// there was one.
	WakeThread(ctx context.Context, req *WakeThreadRequest) (bool, error)
	CreateThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	GetThread(ctx context.Context, req GetThreadRequest) (*conversation.Thread, error)
	// GetThreadOwned reads a thread within a tenant whoever owns it, so the
	// application can act in a conversation on its owner's behalf. It is
	// never reached from a request: a person reads threads through GetThread,
	// scoped to themselves.
	GetThreadOwned(ctx context.Context, req GetThreadOwnedRequest) (*conversation.Thread, error)
	ListThreadAgentsByIDs(
		ctx context.Context,
		req ListThreadAgentsByIDsRequest,
	) ([]*conversation.Thread, error)
	ListThreads(ctx context.Context, req ListThreadsRequest) (*ThreadPage, error)
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
	// MarkThreadTainted writes the thread's taint and, the first time, when it
	// became tainted. It leaves the thread's version alone.
	MarkThreadTainted(ctx context.Context, req MarkThreadTaintedRequest) error
	// UpdateThreadContext writes how full the conversation's context is and
	// whether it compacts itself. It leaves the thread's version alone: it
	// is the system's bookkeeping, not an edit a person could conflict with.
	UpdateThreadContext(ctx context.Context, req UpdateThreadContextRequest) error
	AddSavedMemories(ctx context.Context, req AddSavedMemoriesRequest) error
}
