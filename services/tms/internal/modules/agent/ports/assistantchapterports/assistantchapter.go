package assistantchapterports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ChaptersResource = "assistant_chapters"

type ChapterPhase string

const (
	ChapterPhaseOpened    = ChapterPhase("Opened")
	ChapterPhaseAnswered  = ChapterPhase("Answered")
	ChapterPhaseRequested = ChapterPhase("Requested")
)

type ChapterTurnRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ThreadID   pulid.ID              `json:"threadId"`
	UserID     pulid.ID              `json:"userId"`
	TurnID     pulid.ID              `json:"turnId"`
	Phase      ChapterPhase          `json:"phase"`
}

type ChapterScheduler interface {
	Cue(ctx context.Context, req *ChapterTurnRequest)
}

type ChapterPlan struct {
	Turn ChapterTurnRequest `json:"turn"`

	Skip string `json:"skip,omitempty"`

	ChapterID pulid.ID `json:"chapterId,omitempty"`
	Remove    bool     `json:"remove,omitempty"`

	Opens          bool                                   `json:"opens,omitempty"`
	Decides        bool                                   `json:"decides,omitempty"`
	Fallback       string                                 `json:"fallback,omitempty"`
	FallbackSource conversationchapter.ChapterTitleSource `json:"fallbackSource,omitempty"`
	ModelSource    conversationchapter.ChapterTitleSource `json:"modelSource,omitempty"`
	Evidence       []string                               `json:"evidence,omitempty"`
	// NameEarlier is the turn of the chapter before, still named after its
	// question: a chapter shows only once there is a second, so the first
	// is named when the next one opens.
	NameEarlier pulid.ID `json:"nameEarlier,omitempty"`

	Request *llmports.StructuredCompletionRequest `json:"request,omitempty"`
}

func (p *ChapterPlan) Skipped() bool { return p.Skip != "" }

func (p *ChapterPlan) NeedsModel() bool { return p.Skip == "" && p.Request != nil }

type ChapterOutcome struct {
	ChapterID pulid.ID `json:"chapterId,omitempty"`
	Opened    bool     `json:"opened,omitempty"`
	Retitled  bool     `json:"retitled,omitempty"`
	Removed   bool     `json:"removed,omitempty"`
	Fallback  bool     `json:"fallback,omitempty"`
}

type ListChaptersRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetChapterRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ChapterForTurnRequest struct {
	ThreadID   pulid.ID
	TurnID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type RetitleChapterRequest struct {
	ID          pulid.ID
	ThreadID    pulid.ID
	TenantInfo  pagination.TenantInfo
	Title       string
	TitleSource conversationchapter.ChapterTitleSource
	// Version, when set, refuses the rename if the chapter changed since the
	// person read it; without it a person's own title is never replaced.
	Version *int64
}

type DeleteChapterRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	// KeepPerson leaves a chapter the person named, for the clean-ups that
	// remove what a model or a question opened.
	KeepPerson bool
}

type TurnOfMessageRequest struct {
	ThreadID   pulid.ID
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
}

type ChapterRepository interface {
	List(ctx context.Context, req ListChaptersRequest) ([]*conversationchapter.Chapter, error)
	Get(ctx context.Context, req *GetChapterRequest) (*conversationchapter.Chapter, error)
	ForTurn(ctx context.Context, req *ChapterForTurnRequest) (*conversationchapter.Chapter, error)
	Count(ctx context.Context, req ListChaptersRequest) (int, error)
	Create(
		ctx context.Context,
		chapter *conversationchapter.Chapter,
	) (*conversationchapter.Chapter, error)
	Retitle(ctx context.Context, req *RetitleChapterRequest) (*conversationchapter.Chapter, error)
	Delete(ctx context.Context, req *DeleteChapterRequest) (bool, error)
	TurnOfMessage(ctx context.Context, req *TurnOfMessageRequest) (pulid.ID, error)
}

type ThreadChaptersRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type OpenChapterRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
}

type RenameChapterRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	UserID     pulid.ID
	Title      string
	Version    *int64
	TenantInfo pagination.TenantInfo
}

type RemoveChapterRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type ChapterSpan struct {
	Chapter *conversationchapter.Chapter
	// From is the sequence of the chapter's first message, and Through that
	// of its last, nil while the chapter runs to the end of the conversation.
	From    int
	Through *int
}

type ChapterSpanRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type AssistantChapterService interface {
	List(ctx context.Context, req *ThreadChaptersRequest) ([]*conversationchapter.Chapter, error)
	Open(ctx context.Context, req *OpenChapterRequest) (*conversationchapter.Chapter, error)
	Rename(ctx context.Context, req *RenameChapterRequest) (*conversationchapter.Chapter, error)
	Remove(ctx context.Context, req *RemoveChapterRequest) error
	Span(ctx context.Context, req *ChapterSpanRequest) (*ChapterSpan, error)
	Plan(ctx context.Context, req *ChapterTurnRequest) (*ChapterPlan, error)
	Apply(
		ctx context.Context,
		plan *ChapterPlan,
		result *llmports.StructuredCompletionResult,
	) (*ChapterOutcome, error)
}
