// Package modeladapter translates a provider-agnostic completion request into
// whatever wire protocol a configured endpoint speaks, and normalizes the reply
// back. Adapters are keyed on protocol rather than vendor, so a single
// OpenAI-chat adapter serves every runtime that exposes that shape.
package modeladapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

// Request is one completion, already rendered to text. Adapters do not see the
// caller's domain types; assembling the prompt is the router's job so every
// provider receives byte-identical instructions.
type Request struct {
	System      string
	UserContent string
	// OutputSchema is the JSON Schema the reply must satisfy. Nil requests
	// freeform text. How the schema is enforced — server-side, as a JSON-mode
	// hint, or by prompting alone — depends on the provider's declared mode.
	OutputSchema map[string]any
	// SchemaName labels the schema for the protocols that require a name.
	SchemaName string
	MaxTokens  int
}

// Response is a normalized reply. Text is the raw model output; the router is
// responsible for parsing and repairing it, because how much repair is warranted
// depends on the provider's structured-output mode rather than its protocol.
type Response struct {
	Text            string
	ModelIdentifier string
	InputTokens     int
	OutputTokens    int
	// Refused reports that the model declined the request outright, which is a
	// business outcome rather than a transport failure and must not be retried.
	Refused bool
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
	Request  *Request
}

// Registry resolves an adapter for a provider's protocol.
type Registry struct {
	byKind map[aiprovider.Kind]Adapter
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
	for _, adapter := range adapters {
		byKind[adapter.Kind()] = adapter
	}

	return &Registry{byKind: byKind}
}

// Get returns the adapter for kind.
func (r *Registry) Get(kind aiprovider.Kind) (Adapter, error) {
	adapter, ok := r.byKind[kind]
	if !ok {
		return nil, fmt.Errorf("no adapter for provider kind %q", kind)
	}

	return adapter, nil
}
