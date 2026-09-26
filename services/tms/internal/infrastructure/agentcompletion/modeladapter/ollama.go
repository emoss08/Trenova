package modeladapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
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
	Tools    []ollamaTool    `json:"tools,omitempty"`
	// Format carries the JSON Schema directly, not wrapped in the OpenAI
	// json_schema envelope.
	Format  map[string]any `json:"format,omitempty"`
	Stream  bool           `json:"stream"`
	Options *ollamaOptions `json:"options,omitempty"`
	// Think asks a thinking model to reason first and report it separately.
	Think bool `json:"think,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	Thinking  string           `json:"thinking,omitempty"`
	// ToolName identifies which tool a tool-role message answers, since this
	// protocol carries no call id to pair on.
	ToolName string `json:"tool_name,omitempty"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ollamaToolCall struct {
	Function ollamaToolCallFunc `json:"function"`
}

// ollamaToolCallFunc carries arguments as an object, unlike the OpenAI protocol's
// JSON-encoded string.
type ollamaToolCallFunc struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ollamaOptions struct {
	NumPredict int `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model           string        `json:"model"`
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	DoneReason      string        `json:"done_reason"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
	Error           string        `json:"error"`
}

func (a ollamaAdapter) Stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
) (*Response, error) {
	body := ollamaRequest{
		Model:    call.Provider.Model,
		Messages: toOllamaMessages(call.Request.System, call.Request.Messages),
		Tools:    toOllamaTools(call.Request.Tools),
		Stream:   true,
		Think:    call.reasoning().Enabled(),
	}
	if call.Request.MaxTokens > 0 {
		body.Options = &ollamaOptions{NumPredict: call.Request.MaxTokens}
	}

	stream, err := postStream(
		ctx,
		call,
		call.Provider.ResolvedBaseURL()+"/api/chat",
		bearerHeaders(call.APIKey),
		body,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var (
		text     strings.Builder
		thinking strings.Builder
		final    ollamaResponse
		calls    []ollamaToolCall
	)

	err = readNDJSON(stream, func(line []byte) error {
		var chunk ollamaResponse
		if err := sonic.Unmarshal(line, &chunk); err != nil {
			return fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Error != "" {
			return streamError("server_error", chunk.Error)
		}

		if chunk.Message.Thinking != "" {
			thinking.WriteString(chunk.Message.Thinking)
			call.think(chunk.Message.Thinking)
		}
		if chunk.Message.Content != "" {
			text.WriteString(chunk.Message.Content)
			sink(chunk.Message.Content)
		}
		// Tool calls arrive on whichever chunk the model finished deciding them,
		// usually one of the last, and never repeat.
		calls = append(calls, chunk.Message.ToolCalls...)
		final.Model = stringutils.FirstNonEmpty(final.Model, chunk.Model)
		if chunk.Done {
			final.PromptEvalCount = chunk.PromptEvalCount
			final.EvalCount = chunk.EvalCount
			final.DoneReason = chunk.DoneReason
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	reply, reasoning := mergeInlineThinking(text.String(), textReasoning(thinking.String()))

	return &Response{
		Text:            reply,
		ToolCalls:       fromOllamaToolCalls(calls),
		ModelIdentifier: stringutils.FirstNonEmpty(final.Model, call.Provider.Model),
		InputTokens:     final.PromptEvalCount,
		OutputTokens:    final.EvalCount,
		Refused:         false,
		Truncated:       final.DoneReason == "length",
		Reasoning:       reasoning,
	}, nil
}

func (a ollamaAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := ollamaRequest{
		Model:    call.Provider.Model,
		Messages: toOllamaMessages(call.Request.System, call.Request.Messages),
		Tools:    toOllamaTools(call.Request.Tools),
		// A streamed reply arrives as newline-delimited objects, which would not
		// decode into a single response.
		Stream: false,
		Think:  call.reasoning().Enabled(),
	}

	if call.Request.MaxTokens > 0 {
		body.Options = &ollamaOptions{NumPredict: call.Request.MaxTokens}
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		len(call.Request.Tools) == 0 &&
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
		bearerHeaders(call.APIKey),
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	reply, reasoning := mergeInlineThinking(
		envelope.Message.Content,
		textReasoning(envelope.Message.Thinking),
	)

	return &Response{
		Text:            reply,
		ToolCalls:       fromOllamaToolCalls(envelope.Message.ToolCalls),
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.PromptEvalCount,
		OutputTokens:    envelope.EvalCount,
		// Ollama has no refusal signal; an empty body with a load failure surfaces
		// as a transport error instead.
		Refused:   false,
		Truncated: envelope.DoneReason == "length",
		Reasoning: reasoning,
	}, nil
}

func toOllamaMessages(system string, messages []Message) []ollamaMessage {
	out := make([]ollamaMessage, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		out = append(out, ollamaMessage{Role: "system", Content: system})
	}

	for _, msg := range messages {
		switch msg.Role {
		case RoleTool:
			out = append(out, ollamaMessage{
				Role:     "tool",
				Content:  msg.Content,
				ToolName: msg.ToolName,
			})
		case RoleAssistant:
			message := ollamaMessage{
				Role:      "assistant",
				Content:   msg.Content,
				ToolCalls: toOllamaToolCalls(msg.ToolCalls),
			}
			if msg.Reasoning != nil {
				message.Thinking = msg.Reasoning.Text
			}
			out = append(out, message)
		default:
			out = append(out, ollamaMessage{Role: "user", Content: msg.Content})
		}
	}

	return out
}

func toOllamaToolCalls(calls []ToolCall) []ollamaToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]ollamaToolCall, 0, len(calls))
	for _, call := range calls {
		args := call.Arguments
		if args == nil {
			args = map[string]any{}
		}
		out = append(out, ollamaToolCall{
			Function: ollamaToolCallFunc{Name: call.Name, Arguments: args},
		})
	}

	return out
}

func toOllamaTools(tools []ToolSpec) []ollamaTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]ollamaTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, ollamaTool{
			Type: "function",
			Function: ollamaToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  toolschema.ForModel(tool.Parameters),
			},
		})
	}

	return out
}

// fromOllamaToolCalls synthesizes an id per call. The protocol supplies none, and
// the loop needs something stable to pair a result back to its request. The id
// comes from the call's position, so the first call of every response gets the
// same one; it is marked as synthesized and the loop mints its own in its place.
func fromOllamaToolCalls(calls []ollamaToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]ToolCall, 0, len(calls))
	for idx, call := range calls {
		args := call.Function.Arguments
		if args == nil {
			args = map[string]any{}
		}
		out = append(out, ToolCall{
			ID:            fmt.Sprintf("ollama_call_%d_%s", idx, call.Function.Name),
			SynthesizedID: true,
			Name:          call.Function.Name,
			Arguments:     args,
		})
	}

	return out
}
