package modeladapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/stringutils"
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
	StreamOptions  *chatStreamOptions  `json:"stream_options,omitempty"`
	// ReasoningEffort is sent only when the provider is configured to reason;
	// a model without reasoning rejects the parameter with a 400.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// chatStreamOptions asks for a final usage chunk. Without it a streamed reply
// carries no token counts, and usage is what the AI log is for.
type chatStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls,omitempty"`
	// ReasoningContent is read, never sent: DeepSeek-shaped servers return the
	// chain of thought here and reject a request that echoes it back.
	ReasoningContent string `json:"reasoning_content,omitempty"`
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
	PromptTokens            int `json:"prompt_tokens"`
	CompletionTokens        int `json:"completion_tokens"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
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
	body.ReasoningEffort = call.reasoning().Wire()

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

	text, toolCalls, refused, truncated := firstChatResult(&envelope)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.PromptTokens,
		OutputTokens:    envelope.Usage.CompletionTokens,
		Refused:         refused,
		Truncated:       truncated,
		Reasoning:       firstChatReasoning(&envelope),
		ReasoningTokens: envelope.Usage.CompletionTokensDetails.ReasoningTokens,
	}, nil
}

// firstChatReasoning reads the chain of thought off the first usable choice.
func firstChatReasoning(resp *chatResponse) *ReasoningTrace {
	for idx := range resp.Choices {
		if trace := textReasoning(resp.Choices[idx].Message.ReasoningContent); trace != nil {
			return trace
		}
	}

	return nil
}

// chatStreamChunk is one streamed delta. Tool calls arrive as fragments keyed by
// index, with the id and name on the first fragment and the arguments spread
// across the rest as pieces of one JSON string.
type chatStreamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
}

type chatToolCallBuffer struct {
	id        string
	name      string
	arguments strings.Builder
}

func (a openAIChatAdapter) Stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
) (*Response, error) {
	body := chatRequest{
		Model:         call.Provider.Model,
		MaxTokens:     call.Request.MaxTokens,
		Messages:      toChatMessages(call.Request.System, call.Request.Messages),
		Tools:         toChatTools(call.Request.Tools),
		Stream:        true,
		StreamOptions: &chatStreamOptions{IncludeUsage: true},
	}
	body.ResponseFormat = chatResponseFormatFor(call)
	body.ReasoningEffort = call.reasoning().Wire()

	stream, err := postStream(
		ctx,
		call,
		call.Provider.ResolvedBaseURL()+"/chat/completions",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var (
		text      strings.Builder
		thinking  strings.Builder
		model     string
		usage     chatUsage
		refused   bool
		truncated bool
		// Buffers are keyed by index, which is the protocol's own key for a
		// call's fragments — but a fragment carrying a different id at an
		// index already in use is a new call, not a continuation. Some
		// providers put every call at index 0 and tell them apart by id alone,
		// and keyed on index alone they merged into one mangled buffer.
		buffers = map[string]*chatToolCallBuffer{}
		active  = map[int]string{}
		order   []string
	)

	err = readSSE(stream, func(_, data string) error {
		if data == "[DONE]" {
			return nil
		}

		var chunk chatStreamChunk
		if err := sonic.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("decode stream chunk: %w", err)
		}

		model = stringutils.FirstNonEmpty(model, chunk.Model)
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}

		for idx := range chunk.Choices {
			choice := &chunk.Choices[idx]
			switch choice.FinishReason {
			case "content_filter":
				refused = true
			case "length":
				truncated = true
			}
			if choice.Delta.ReasoningContent != "" {
				thinking.WriteString(choice.Delta.ReasoningContent)
				call.think(choice.Delta.ReasoningContent)
			}
			if choice.Delta.Content != "" {
				text.WriteString(choice.Delta.Content)
				sink(choice.Delta.Content)
			}
			for _, fragment := range choice.Delta.ToolCalls {
				key, ok := active[fragment.Index]
				if !ok || (fragment.ID != "" && buffers[key].id != "" && buffers[key].id != fragment.ID) {
					key = fmt.Sprintf("%d/%s", fragment.Index, fragment.ID)
					active[fragment.Index] = key
					buffers[key] = &chatToolCallBuffer{}
					order = append(order, key)
				}
				buffer := buffers[key]
				buffer.id = stringutils.FirstNonEmpty(buffer.id, fragment.ID)
				buffer.name = stringutils.FirstNonEmpty(buffer.name, fragment.Function.Name)
				buffer.arguments.WriteString(fragment.Function.Arguments)
			}
		}

		return nil
	})
	if err != nil {
		return nil, interrupted(err, model)
	}

	toolCalls := make([]ToolCall, 0, len(order))
	for position, key := range order {
		buffer := buffers[key]
		arguments, argumentsErr := decodeArguments(buffer.arguments.String())
		toolCalls = append(toolCalls, ToolCall{
			ID:             stringutils.FirstNonEmpty(buffer.id, fmt.Sprintf("call_%d", position)),
			Name:           buffer.name,
			Arguments:      arguments,
			ArgumentsError: argumentsErr,
		})
	}
	if len(toolCalls) == 0 {
		toolCalls = nil
	}

	return &Response{
		Text:            text.String(),
		ToolCalls:       toolCalls,
		ModelIdentifier: stringutils.FirstNonEmpty(model, call.Provider.Model),
		InputTokens:     usage.PromptTokens,
		OutputTokens:    usage.CompletionTokens,
		Refused:         refused,
		Truncated:       truncated,
		Reasoning:       textReasoning(thinking.String()),
		ReasoningTokens: usage.CompletionTokensDetails.ReasoningTokens,
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

// firstChatResult picks the first usable choice: its text, its tool calls,
// whether it was refused, and whether it was cut off by the output limit.
func firstChatResult(resp *chatResponse) (string, []ToolCall, bool, bool) {
	for idx := range resp.Choices {
		choice := &resp.Choices[idx]
		if choice.FinishReason == "content_filter" {
			return "", nil, true, false
		}

		toolCalls := fromChatToolCalls(choice.Message.ToolCalls)
		if len(toolCalls) > 0 || strings.TrimSpace(choice.Message.Content) != "" {
			return choice.Message.Content, toolCalls, false, choice.FinishReason == "length"
		}
	}

	return "", nil, false, false
}

func fromChatToolCalls(calls []chatToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		args, argsErr := decodeArguments(call.Function.Arguments)
		out = append(out, ToolCall{
			ID:             call.ID,
			Name:           call.Function.Name,
			Arguments:      args,
			ArgumentsError: argsErr,
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
