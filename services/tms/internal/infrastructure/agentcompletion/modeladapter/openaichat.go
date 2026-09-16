package modeladapter

import (
	"context"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type openAIChatAdapter struct{}

// NewOpenAIChatAdapter speaks the OpenAI Chat Completions API, which is the de
// facto interchange format: vLLM, SGLang, LM Studio, llama.cpp, OpenRouter, Groq,
// Together, Fireworks, DeepInfra and Bedrock's chat-completions endpoint all
// accept it. What varies between them is not the request shape but how much of
// response_format they honour, which is why the provider declares its own
// structured-output mode instead of the adapter assuming one.
func NewOpenAIChatAdapter() Adapter { return openAIChatAdapter{} }

func (openAIChatAdapter) Kind() aiprovider.Kind { return aiprovider.KindOpenAIChat }

type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	MaxTokens      int                 `json:"max_completion_tokens,omitempty"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
	Tools          []chatTool          `json:"tools,omitempty"`
	Stream         bool                `json:"stream"`
}

type chatMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls,omitempty"`
	// ToolCallID pairs a tool-role message with the call it answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatToolCallFunc `json:"function"`
}

// chatToolCallFunc carries arguments as a JSON-encoded string rather than an
// object, which is the one place this protocol differs from Ollama's.
type chatToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"`
}

type chatJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type chatResponse struct {
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (a openAIChatAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := chatRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		Messages:  toChatMessages(call.Request.System, call.Request.Messages),
		Tools:     toChatTools(call.Request.Tools),
		Stream:    false,
	}
	body.ResponseFormat = chatResponseFormatFor(call)

	var envelope chatResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/chat/completions",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	text, toolCalls, refused := firstChatResult(&envelope)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: firstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.PromptTokens,
		OutputTokens:    envelope.Usage.CompletionTokens,
		Refused:         refused,
	}, nil
}

func toChatMessages(system string, messages []Message) []chatMessage {
	out := make([]chatMessage, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		out = append(out, chatMessage{Role: "system", Content: system})
	}

	for _, msg := range messages {
		switch msg.Role {
		case RoleTool:
			out = append(out, chatMessage{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
			})
		case RoleAssistant:
			out = append(out, chatMessage{
				Role:      "assistant",
				Content:   msg.Content,
				ToolCalls: toChatToolCalls(msg.ToolCalls),
			})
		default:
			out = append(out, chatMessage{Role: "user", Content: msg.Content})
		}
	}

	return out
}

func toChatToolCalls(calls []ToolCall) []chatToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]chatToolCall, 0, len(calls))
	for _, call := range calls {
		encoded, err := sonic.Marshal(call.Arguments)
		if err != nil {
			encoded = []byte("{}")
		}
		out = append(out, chatToolCall{
			ID:       call.ID,
			Type:     "function",
			Function: chatToolCallFunc{Name: call.Name, Arguments: string(encoded)},
		})
	}

	return out
}

func toChatTools(tools []ToolSpec) []chatTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]chatTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, chatTool{
			Type: "function",
			Function: chatToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	return out
}

// chatResponseFormatFor asks for exactly as much enforcement as the endpoint was
// declared to support. Sending a json_schema to a server that ignores it is worse
// than not sending one, because the reply comes back unconstrained with no error
// to signal that the schema was dropped.
func chatResponseFormatFor(call *Call) *chatResponseFormat {
	// Constraining the reply to a schema while also offering tools leaves the
	// model no way to express a tool call, and most endpoints resolve the
	// contradiction by dropping one silently.
	if call.Request.OutputSchema == nil || len(call.Request.Tools) > 0 {
		return nil
	}

	switch call.Provider.StructuredOutputMode {
	case aiprovider.StructuredOutputJSONSchema:
		return &chatResponseFormat{
			Type: "json_schema",
			JSONSchema: &chatJSONSchema{
				Name:   call.Request.SchemaName,
				Schema: call.Request.OutputSchema,
				Strict: true,
			},
		}
	case aiprovider.StructuredOutputJSONMode:
		return &chatResponseFormat{Type: "json_object"}
	case aiprovider.StructuredOutputPrompted:
		// The schema rides in the system prompt instead; the reply is repaired
		// and validated on receipt.
		return nil
	default:
		return nil
	}
}

func firstChatResult(resp *chatResponse) (string, []ToolCall, bool) {
	for idx := range resp.Choices {
		choice := &resp.Choices[idx]
		if choice.FinishReason == "content_filter" {
			return "", nil, true
		}

		toolCalls := fromChatToolCalls(choice.Message.ToolCalls)
		if len(toolCalls) > 0 || strings.TrimSpace(choice.Message.Content) != "" {
			return choice.Message.Content, toolCalls, false
		}
	}

	return "", nil, false
}

func fromChatToolCalls(calls []chatToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		args := map[string]any{}
		// A small model sometimes emits arguments that do not parse. An empty
		// argument map lets the tool report a clear validation error, which the
		// model can recover from, rather than failing the whole turn here.
		if trimmed := strings.TrimSpace(call.Function.Arguments); trimmed != "" {
			_ = sonic.Unmarshal([]byte(trimmed), &args)
		}
		out = append(out, ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: args,
		})
	}

	return out
}

func bearer(apiKey string) string {
	if strings.TrimSpace(apiKey) == "" {
		return ""
	}

	return "Bearer " + apiKey
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
