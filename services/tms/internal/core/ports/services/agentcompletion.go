package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/shopspring/decimal"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrModelSchemaValidation = errors.New("model output failed schema validation")
	// ErrNoProviderConfigured reports that no enabled provider is assigned to the
	// requested task. It is distinct from a call failure: nothing was attempted.
	ErrNoProviderConfigured = errors.New("no AI provider is configured for this task")
	// ErrProvidersResting reports that every provider the request could use
	// is paused after repeated failures, so nothing was attempted.
	ErrProvidersResting            = errors.New("every usable AI provider is resting after repeated failures")
	ErrRequiredProviderUnavailable = errors.New("the required AI provider cannot serve this task")
	// ErrProviderCapReached reports a provider that has spent its monthly
	// cap: every one the request could use, or one set to stop there.
	ErrProviderCapReached = errors.New("the AI provider has reached its monthly spending cap")
	// ErrProviderBusy reports a provider already taking as many calls at
	// once as it is allowed.
	ErrProviderBusy = errors.New("the AI provider is at its limit of calls at once")
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
	RequireProvider     bool
	// HedgeAfter is for a caller bounded by a short deadline: when the
	// provider asked last has not answered by then, the next in the order is
	// asked too and the first answer wins, so one slow provider does not spend
	// the whole budget. Zero asks the providers one after another.
	HedgeAfter time.Duration
	// Attribution says who the call is for, so its cost lands somewhere.
	Attribution AIUsageAttribution
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
	LatencyMs    int64
	CostUSD      *decimal.Decimal
}

// ChatCompletionRequest is one turn of a tool-using conversation. Unlike the
// structured path it sends the whole exchange, because these APIs are stateless.
type ChatCompletionRequest struct {
	TenantInfo pagination.TenantInfo
	System     string
	// SystemStable is how many leading bytes of System are the same on every
	// turn, which a provider that caches by explicit mark can cache apart from
	// the turn's own part. Zero treats the whole prompt as one.
	SystemStable int
	Messages     []Message
	Tools        []ToolSpec
	MaxTokens    int
	// PreferredProviderID asks for one configured provider first. It is honoured
	// only when that provider is enabled and serves the task; otherwise the usual
	// priority order applies, so a deleted preference never strands an agent.
	PreferredProviderID pulid.ID
	// Attribution says who the call is for, so its cost lands somewhere.
	Attribution AIUsageAttribution
	// ReasoningSink receives the model's thinking as it streams, when the
	// provider lets it through. Optional, and separate from the text sink
	// because thinking is not the reply: it is shown differently and never
	// becomes the message.
	ReasoningSink ChatStreamSink `json:"-"`
	// PinPreferred restricts the turn to the preferred provider when it is
	// usable, instead of trying it first and falling through. A person who
	// picked a model in the composer asked for that model, not for whatever
	// answers when it fails.
	PinPreferred bool
	// RetrySink is told when a provider died partway through a reply and the
	// turn is starting over on another attempt. Whatever reached the text
	// sink before it is being discarded, and the reader should see that.
	RetrySink func(ChatRetryNotice) `json:"-"`
}

// ChatRetryNotice says a reply is starting again after a provider failed
// mid-way. Attempt counts the retries so far, starting at 1.
type ChatRetryNotice struct {
	Attempt  int
	Provider string
	Reason   string
	// Kind says what the retry is: a reply that died partway starting over
	// on another provider, or a busy provider being asked again after a
	// wait. The reader is told different things for each.
	Kind RetryKind
	// WaitSeconds is how long the router is waiting before a busy retry.
	WaitSeconds int
	// MaxAttempts is how many times the provider is asked in all before the
	// router gives up on it, so the reader can be told "attempt 2 of 4".
	MaxAttempts int
}

// RetryKind names why a turn is being retried.
type RetryKind string

const (
	RetryKindRestart RetryKind = "restart"
	RetryKindBusy    RetryKind = "busy"
)

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
	// Reasoning is the model's thinking, when the provider produced any.
	Reasoning *conversation.ReasoningTrace
	// ReasoningTokens is the thinking's share of OutputTokens, where the
	// protocol reports one.
	ReasoningTokens int
	// LatencyMs is how long the answering call took, from request to the last
	// byte. Attempts that fell through to another provider are not counted
	// here; the usage record has them.
	LatencyMs int64
	// CostUSD is what the answering call cost at the provider's configured
	// price, nil when the provider carries none.
	CostUSD     *decimal.Decimal
	OutputLimit int
	CutOffCall  *CutOffToolCall
	// ContextWindow is the window the answering provider is configured with,
	// zero when it is left to be read off the model id.
	ContextWindow int
	// FallbackFrom is the provider asked first when another one answered:
	// the reader is told which model replied and which one didn't.
	FallbackFrom *ChatProviderFailure
}

// ChatProviderFailure is one provider that was asked and didn't answer.
type ChatProviderFailure struct {
	ProviderID pulid.ID `json:"providerId"`
	Name       string   `json:"name"`
	Model      string   `json:"model"`
	// Vendor is the company behind the model, for its mark.
	Vendor string `json:"vendor"`
	// Status is a short word for what went wrong: Overloaded, Timed out,
	// Unavailable, Failed, or Not set up for a model the organization has
	// but has not given the task.
	Status string `json:"status"`
	// Detail says what happened in a line a person can read: "Anthropic
	// returned 529 twice", "No response after 30s".
	Detail string `json:"detail"`
	// HTTPStatus is the provider's status code, when it answered at all, and
	// Attempts how many times it was asked.
	HTTPStatus int `json:"httpStatus,omitempty"`
	Attempts   int `json:"attempts,omitempty"`
}

// ChatProvidersFailedError is returned when every provider in the order was
// asked and none answered. It names each, so the reader is told more than
// that something failed.
type ChatProvidersFailedError struct {
	Failures []ChatProviderFailure
	Err      error
}

func (e *ChatProvidersFailedError) Error() string {
	return fmt.Sprintf("every configured chat provider failed: %v", e.Err)
}

func (e *ChatProvidersFailedError) Unwrap() error {
	return e.Err
}

type CutOffToolCall struct {
	Name string `json:"name,omitempty"`
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
	TenantInfo  pagination.TenantInfo
	ProviderID  pulid.ID
	Handle      string
	Task        aiprovider.Task
	SubmittedAt int64
	Attribution AIUsageAttribution
}

type BackgroundOutcome struct {
	State           BackgroundState
	RawStatus       string
	ModelIdentifier string
	Result          *StructuredCompletionResult
	FailureCode     string
	FailureMessage  string
}

// StructuredCompleter answers one structured question. A service answering a
// person's request takes this rather than the CompletionService: the call runs
// as a workflow on a worker, retried the way the provider's answer says, and
// the request waits for its result.
type StructuredCompleter interface {
	CompleteStructured(
		ctx context.Context,
		req *StructuredCompletionRequest,
	) (*StructuredCompletionResult, error)
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

// ProviderFailure is an error that carries a provider's own verdict on a
// request: the status it answered with, and whether the failure is the
// provider being unavailable rather than the request being wrong. Adapters
// implement it on their transport errors; the assistant reads it to tell the
// person whether their request was refused or the provider could not be
// reached, which are different things to do something about.
type ProviderFailure interface {
	error
	ProviderStatus() int
	ProviderRetryable() bool
}

// ProviderBackoff is a provider failure that said how long to wait before
// asking again, from a Retry-After header or its equivalent. Zero means it
// said nothing.
type ProviderBackoff interface {
	error
	ProviderRetryAfter() time.Duration
}
