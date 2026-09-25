// Package modeladapter translates a provider-agnostic completion request into
// whatever wire protocol a configured endpoint speaks, and normalizes the reply
// back. Adapters are keyed on protocol rather than vendor, so a single
// OpenAI-chat adapter serves every runtime that exposes that shape.
package modeladapter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

// Request is one completion, already rendered to text. Adapters do not see the
// caller's domain types; assembling the prompt is the router's job so every
// provider receives byte-identical instructions.
type Request struct {
	System string
	// Messages is the conversation so far. A one-shot completion sends a single
	// user message; a tool loop sends the whole exchange back each turn, because
	// these APIs are stateless.
	Messages []Message
	// OutputSchema is the JSON Schema the reply must satisfy. Nil requests
	// freeform text. How the schema is enforced — server-side, as a JSON-mode
	// hint, or by prompting alone — depends on the provider's declared mode.
	OutputSchema map[string]any
	// SchemaName labels the schema for the protocols that require a name.
	SchemaName string
	// Tools the model may ask for. Offering tools and demanding a JSON schema at
	// once confuses most endpoints, so callers set one or the other.
	Tools     []ToolSpec
	MaxTokens int
	// Sampling is how adventurously the model may pick its next token. An
	// empty value leaves the endpoint's own defaults alone, which is what
	// every call used to do.
	Sampling Sampling
}

// Response is a normalized reply. Text is the raw model output; the router is
// responsible for parsing and repairing it, because how much repair is warranted
// depends on the provider's structured-output mode rather than its protocol.
type Response struct {
	Text            string
	ModelIdentifier string
	InputTokens     int
	OutputTokens    int
	// ToolCalls is what the model asked to run. A turn can carry both text and
	// tool calls.
	ToolCalls []ToolCall
	// Refused reports that the model declined the request outright, which is a
	// business outcome rather than a transport failure and must not be retried.
	Refused bool
	// Truncated reports that the model stopped because it hit its output limit
	// rather than because it was finished. Every protocol names this
	// differently — finish_reason "length", stop_reason "max_tokens", an
	// incomplete response, done_reason "length" — and none of them used to be
	// read, so a reply cut off by the limit looked exactly like a finished one.
	Truncated bool
	// Reasoning is what the model thought first, when the protocol carried it.
	Reasoning *ReasoningTrace
	// ReasoningTokens is how many of the output tokens were thinking, where
	// the protocol says. Anthropic folds them into output_tokens and reports
	// no split, so zero here does not mean the model did not think.
	ReasoningTokens  int
	CacheReadTokens  int
	CacheWriteTokens int
}

func CacheSeparateFromInput(kind aiprovider.Kind) bool {
	return kind == aiprovider.KindAnthropicMessages
}

// Adapter speaks one wire protocol.
type Adapter interface {
	Kind() aiprovider.Kind
	Complete(ctx context.Context, call *Call) (*Response, error)
}

// Call carries everything an adapter needs for a single request. The HTTP client
// is supplied rather than owned because its dialer enforces the provider's egress
// policy, and that policy must not be reusable across providers.
type Call struct {
	Provider *aiprovider.Provider
	APIKey   string
	Client   *http.Client
	// StreamClient serves streaming calls, which need no whole-request
	// deadline: http.Client.Timeout spans reading the body, so a client that
	// carries one severs a healthy stream partway through a long answer. A
	// stalled stream is caught by an idle guard instead. Falls back to Client
	// when unset.
	StreamClient *http.Client
	// StreamIdle is how long a stream may go silent before it is abandoned.
	// Measured between reads, so a slow model is not a stalled one.
	StreamIdle time.Duration
	Request    *Request
	// Reasoning receives the model's thinking as it streams, when the
	// protocol carries it. Optional; a call without one still reads the
	// thinking into the Response, it just does not show it as it happens.
	Reasoning StreamSink
}

// think hands a piece of thinking to the reasoning sink, if there is one.
func (c *Call) think(delta string) {
	if c.Reasoning != nil && delta != "" {
		c.Reasoning(delta)
	}
}

// reasoning is how hard this call asks the model to think, from the
// provider. An unset effort is off: the column defaults to Off and most
// providers never name it, so treating the empty string as anything else
// would have a provider nobody configured to think counted as thinking.
func (c *Call) reasoning() aiprovider.ReasoningEffort {
	if c.Provider == nil || c.Provider.ReasoningEffort == "" {
		return aiprovider.ReasoningOff
	}

	return c.Provider.ReasoningEffort
}

// defaultStreamIdle is the silence a stream is allowed when nothing configures
// one.
//
// Five minutes, not ninety seconds. A model that streams its reasoning resets
// this on every thinking delta, but one that reasons out of sight sends nothing
// at all until its first answer token, and a hard question on a heavy model can
// hold that silence for minutes. This is a dead-connection check, nothing more:
// waiting five minutes on a socket that has actually died is a far smaller cost
// than severing a live answer that was about to arrive.
const defaultStreamIdle = 5 * time.Minute

// streamHTTPClient is the client a streaming call should use.
func (c *Call) streamHTTPClient() *http.Client {
	if c.StreamClient != nil {
		return c.StreamClient
	}

	return c.Client
}

// streamIdleTimeout is the silence this call tolerates.
func (c *Call) streamIdleTimeout() time.Duration {
	if c.StreamIdle > 0 {
		return c.StreamIdle
	}

	return defaultStreamIdle
}

// Registry resolves an adapter for a provider's protocol.
type Registry struct {
	byKind    map[aiprovider.Kind]Adapter
	embedders map[aiprovider.Kind]Embedder
}

// NewRegistry builds the registry over every protocol this system speaks.
func NewRegistry() *Registry {
	adapters := []Adapter{
		NewAnthropicAdapter(),
		NewOpenAIResponsesAdapter(),
		NewOpenAIChatAdapter(),
		NewOllamaAdapter(),
	}

	byKind := make(map[aiprovider.Kind]Adapter, len(adapters))
	embedders := make(map[aiprovider.Kind]Embedder, len(adapters))
	for _, adapter := range adapters {
		byKind[adapter.Kind()] = adapter
		if embedder, ok := adapter.(Embedder); ok && adapter.Kind().SupportsEmbedding() {
			embedders[adapter.Kind()] = embedder
		}
	}

	return &Registry{byKind: byKind, embedders: embedders}
}

// Get returns the adapter for kind.
func (r *Registry) Get(kind aiprovider.Kind) (Adapter, error) {
	adapter, ok := r.byKind[kind]
	if !ok {
		return nil, fmt.Errorf("no adapter for provider kind %q", kind)
	}

	return adapter, nil
}

// BackgroundState is where a submitted run has got to.
type BackgroundState string

const (
	BackgroundPending   = BackgroundState("Pending")
	BackgroundCompleted = BackgroundState("Completed")
	BackgroundFailed    = BackgroundState("Failed")
)

// BackgroundHandle identifies a run the provider is holding for us.
type BackgroundHandle struct {
	ID              string
	ModelIdentifier string
	Status          string
}

// BackgroundOutcome is one poll of a submitted run.
type BackgroundOutcome struct {
	State BackgroundState
	// RawStatus is the provider's own word for it, kept for the audit trail
	// because every vendor spells these differently.
	RawStatus       string
	ModelIdentifier string
	Response        *Response
	FailureCode     string
	FailureMessage  string
}

// BackgroundRunner is implemented by a protocol that can start work and be asked
// about it later. Extraction of a long document takes minutes, and holding an
// HTTP request open for that is how a worker ends up blocked on a socket.
//
// Only one protocol offers this today. A provider without it is not excluded
// from the work; the caller runs it inline instead and gets its answer at once.
type BackgroundRunner interface {
	Submit(ctx context.Context, call *Call) (*BackgroundHandle, error)
	Poll(ctx context.Context, call *Call, id string) (*BackgroundOutcome, error)
}
