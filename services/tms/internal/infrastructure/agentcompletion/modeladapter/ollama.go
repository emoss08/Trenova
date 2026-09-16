package modeladapter

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type ollamaAdapter struct{}

// NewOllamaAdapter speaks Ollama's native /api/chat rather than its
// OpenAI-compatible endpoint. Ollama does expose /v1/chat/completions, but that
// endpoint silently discards response_format.json_schema (ollama/ollama#10001):
// the request succeeds and returns unconstrained prose, so a schema violation
// surfaces as a parse failure far from its cause. The native endpoint takes the
// schema in `format` and constrains decoding against it, which is the behaviour
// worth having — Ollama is the most common way an organization self-hosts.
func NewOllamaAdapter() Adapter { return ollamaAdapter{} }

func (ollamaAdapter) Kind() aiprovider.Kind { return aiprovider.KindOllama }

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	// Format carries the JSON Schema directly, not wrapped in the OpenAI
	// json_schema envelope.
	Format  map[string]any `json:"format,omitempty"`
	Stream  bool           `json:"stream"`
	Options *ollamaOptions `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	NumPredict int `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model           string        `json:"model"`
	Message         ollamaMessage `json:"message"`
	DoneReason      string        `json:"done_reason"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
}

func (a ollamaAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := ollamaRequest{
		Model: call.Provider.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: call.Request.System},
			{Role: "user", Content: call.Request.UserContent},
		},
		// A streamed reply arrives as newline-delimited objects, which would not
		// decode into a single response.
		Stream: false,
	}

	if call.Request.MaxTokens > 0 {
		body.Options = &ollamaOptions{NumPredict: call.Request.MaxTokens}
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.Format = schema
	}

	var envelope ollamaResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/api/chat",
		// Ollama itself takes no credential, but the same protocol is served
		// behind authenticating reverse proxies, so a key is sent when present.
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	return &Response{
		Text:            envelope.Message.Content,
		ModelIdentifier: firstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.PromptEvalCount,
		OutputTokens:    envelope.EvalCount,
		// Ollama has no refusal signal; an empty body with a load failure surfaces
		// as a transport error instead.
		Refused: false,
	}, nil
}
