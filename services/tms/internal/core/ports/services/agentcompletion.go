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
	// PreferredProviderID asks for one configured provider first. It is honoured
	// only when that provider is enabled and serves the task; otherwise the usual
	// priority order applies, so a deleted preference never strands a caller.
	PreferredProviderID pulid.ID
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
	// Truncated reports that the provider stopped partway through the reply.
	// Text holds what arrived, which is worth keeping: a reader who watched
	// half an answer appear should not be left with nothing, and the half that
	// arrived is usually the part that answered the question.
	Truncated bool
}

// ChatStreamSink receives reply text as the model produces it. It is a preview
// of the result's Text, delivered in order; the result is still authoritative.
type ChatStreamSink func(delta string)

// BackgroundState is where a deferred call stands. It is deliberately smaller
// than the set of statuses any one protocol reports: a caller only needs to know
// whether to wait, read the answer, or give up.
type BackgroundState string

const (
	BackgroundPending   BackgroundState = "pending"
	BackgroundCompleted BackgroundState = "completed"
	BackgroundFailed    BackgroundState = "failed"
)

// BackgroundSubmission is the outcome of handing a structured call to a provider
// for deferred execution.
type BackgroundSubmission struct {
	// Handle identifies the deferred call to the provider that accepted it, and
	// is meaningful only to that provider. It is empty when no candidate could
	// defer: Result then carries an answer produced inline and there is nothing
	// to poll, so a caller never needs a second code path for the providers whose
	// protocol has no background mode.
	Handle          string
	ProviderID      pulid.ID
	ProviderKind    aiprovider.Kind
	ModelIdentifier string
	RawStatus       string
	Result          *StructuredCompletionResult
}

// BackgroundPollRequest asks after a submission. ProviderID is required because
// a handle belongs to the endpoint that issued it; asking a different provider
// about it would at best 404 and at worst read someone else's call.
type BackgroundPollRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	Handle     string
}

type BackgroundOutcome struct {
	State           BackgroundState
	RawStatus       string
	ModelIdentifier string
	Result          *StructuredCompletionResult
	FailureCode     string
	FailureMessage  string
}

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
	// SubmitBackground hands a structured call to the first candidate whose
	// protocol can run it asynchronously, which is what keeps a long extraction
	// off an activity's heartbeat. When no candidate can defer, the call is run
	// inline and the answer comes back on the submission.
	SubmitBackground(
		ctx context.Context,
		req *StructuredCompletionRequest,
	) (*BackgroundSubmission, error)
	// PollBackground reports where a submission stands. A provider that has since
	// been deleted or disabled fails the poll rather than silently waiting.
	PollBackground(
		ctx context.Context,
		req *BackgroundPollRequest,
	) (*BackgroundOutcome, error)
}
