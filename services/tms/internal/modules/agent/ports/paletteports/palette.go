package paletteports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Question is what a person typed into the command palette and where they
// were when they typed it.
type Question struct {
	Content  string                        `json:"content"`
	Page     *agentpagecontext.PageContext `json:"page,omitempty"`
	Mentions []agentpagecontext.EntityRef  `json:"mentions,omitempty"`
}

type AskRequest struct {
	Question   Question
	TenantInfo pagination.TenantInfo
}

// Ask is a question handed to a worker: its id, and where its answer streams.
type Ask struct {
	ID        pulid.ID `json:"askId"`
	StreamURL string   `json:"streamUrl"`
}

// Answer is how a question came out, kept as its workflow's result so "Open
// in Desk" can carry it over.
type Answer struct {
	Question Question             `json:"question"`
	Answer   *llmports.NovaAnswer `json:"answer,omitempty"`
	Failure  string               `json:"failure,omitempty"`
}

// AnsweredEvent ends an answer's stream.
type AnsweredEvent struct {
	AskID pulid.ID `json:"askId"`
}

// Kept is the conversation a question was carried into. Unanswered is the
// question when it had no answer to carry, for the client to ask in the Desk.
type Kept struct {
	Thread     *conversation.Thread `json:"thread"`
	Unanswered string               `json:"unanswered,omitempty"`
}

// AskRef names one of a person's questions.
type AskRef struct {
	AskID      pulid.ID
	TenantInfo pagination.TenantInfo
}

type RelayRequest struct {
	Ref     AskRef
	Cursor  string
	OnFrame assistantstreamports.TurnFrameFunc
}

// PaletteService answers command palette questions on the Nova harness, and
// carries a finished one into the Desk when the person wants to go on.
type PaletteService interface {
	Ask(
		ctx context.Context,
		req *AskRequest,
		actor *permissionports.RequestActor,
	) (*Ask, error)
	Relay(ctx context.Context, req *RelayRequest) error
	Stop(ctx context.Context, ref AskRef) error
	Keep(
		ctx context.Context,
		ref AskRef,
		actor *permissionports.RequestActor,
	) (*Kept, error)
}
