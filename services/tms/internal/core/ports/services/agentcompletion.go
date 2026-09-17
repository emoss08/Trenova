package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrModelSchemaValidation = errors.New("model output failed schema validation")
	// ErrNoProviderConfigured reports that no enabled provider is assigned to the
	// requested task. It is distinct from a call failure: nothing was attempted.
	ErrNoProviderConfigured = errors.New("no AI provider is configured for this task")
)

type ContextSection struct {
	Title   string
	Trusted bool
	Content string
}

type DelimitedContext struct {
	Sections []ContextSection
}

type StructuredCompletionRequest struct {
	TenantInfo pagination.TenantInfo
	// Task selects which configured providers may serve this call. Leaving it
	// empty routes to the general-purpose pool.
	Task         aiprovider.Task
	System       string
	Context      DelimitedContext
	OutputSchema map[string]any
	// SchemaName labels the schema for the protocols that require one.
	SchemaName string
	MaxTokens  int
}

type StructuredCompletionResult struct {
	Text            string
	ModelIdentifier string
	InputTokens     int
	OutputTokens    int
	// ProviderID and ProviderKind attribute the call to the endpoint that served
	// it, which is what makes per-provider spend and reliability reportable once
	// several are configured.
	ProviderID   pulid.ID
	ProviderKind aiprovider.Kind
}

// ChatCompletionRequest is one turn of a tool-using conversation. Unlike the
// structured path it sends the whole exchange, because these APIs are stateless.
type ChatCompletionRequest struct {
	TenantInfo pagination.TenantInfo
	System     string
	Messages   []Message
	Tools      []ToolSpec
	MaxTokens  int
	// PreferredProviderID asks for one configured provider first. It is honoured
	// only when that provider is enabled and serves the task; otherwise the usual
	// priority order applies, so a deleted preference never strands an agent.
	PreferredProviderID pulid.ID
}

// ChatCompletionResult is a turn's reply, which may ask for tools, say
// something, or both.
type ChatCompletionResult struct {
	Text            string
	ToolCalls       []ToolCall
	ModelIdentifier string
	InputTokens     int
	OutputTokens    int
	ProviderID      pulid.ID
	ProviderKind    aiprovider.Kind
}

// ChatStreamSink receives reply text as the model produces it. It is a preview
// of the result's Text, delivered in order; the result is still authoritative.
type ChatStreamSink func(delta string)

type CompletionService interface {
	CompleteStructured(
		ctx context.Context,
		req *StructuredCompletionRequest,
	) (*StructuredCompletionResult, error)
	// CompleteChat runs one conversational turn with tools. It does not loop;
	// driving the loop is the assistant service's job, since only that layer knows
	// which tools may run and which need a person.
	CompleteChat(
		ctx context.Context,
		req *ChatCompletionRequest,
	) (*ChatCompletionResult, error)
	// StreamChat is CompleteChat with the reply text delivered to sink as it
	// arrives. A provider whose protocol cannot stream delivers the whole text at
	// once, so callers need no second path.
	StreamChat(
		ctx context.Context,
		req *ChatCompletionRequest,
		sink ChatStreamSink,
	) (*ChatCompletionResult, error)
}
