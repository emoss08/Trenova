package llmports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NovaTool is one read a conversation task may make. Its caller binds it to
// the person asking, so every call reads only what they may read.
type NovaTool interface {
	Spec() kernelports.ToolSpec
	Call(ctx context.Context, arguments map[string]any) NovaToolResult
}

// NovaToolResult is what a read returned: what the model reads, a line for
// the person watching, and the records it described.
type NovaToolResult struct {
	Content string                       `json:"content"`
	Failed  bool                         `json:"failed,omitempty"`
	Summary string                       `json:"summary,omitempty"`
	Records []agentpagecontext.EntityRef `json:"records,omitempty"`
	// Refused says the call was turned away before anything was read: the
	// person may not read the kind, or the arguments did not fit. A refused
	// call is also Failed.
	Refused bool `json:"refused,omitempty"`
	// Read is the read the call went to, and Params what that read was asked,
	// when the tool reads through another (the palette's get_record runs
	// customer_get). They are what the call's record keeps to explain it.
	Read   string         `json:"read,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

// NovaConversation is one question Nova answers from a few reads, with the
// answer streamed as it is written. It is not a Desk turn: nothing is saved
// to a conversation and no Desk guard or prompt is involved.
type NovaConversation struct {
	Name         aitask.Task
	Feature      aiusage.Feature
	Instructions string
	Question     string
	Context      DelimitedContext
	Tools        []NovaTool
	// MaxCalls bounds the model calls; the last one is asked to answer
	// without reading further.
	MaxCalls   int
	MaxTokens  int
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	// OwnerKind and OwnerID name what the conversation answers (a palette
	// question), so its model calls are kept with it on the AI audit trail.
	OwnerKind kernelports.RunStepOwnerKind
	OwnerID   pulid.ID
}

// NovaAnswer is the finished answer and the records it read.
type NovaAnswer struct {
	Text       string                       `json:"text"`
	Records    []agentpagecontext.EntityRef `json:"records,omitempty"`
	Model      string                       `json:"model,omitempty"`
	ProviderID pulid.ID                     `json:"providerId,omitzero"`
	Truncated  bool                         `json:"truncated,omitempty"`
}

// NovaEvents hears a conversation as it happens, in order.
type NovaEvents interface {
	Delta(text string)
	Message(content string, calls []ToolCall, model string)
	ToolStarted(call ToolCall)
	ToolFinished(call ToolCall, result NovaToolResult)
}

// TaskConverser answers a conversation task on a worker.
type TaskConverser interface {
	Converse(ctx context.Context, talk *NovaConversation, events NovaEvents) (*NovaAnswer, error)
}
