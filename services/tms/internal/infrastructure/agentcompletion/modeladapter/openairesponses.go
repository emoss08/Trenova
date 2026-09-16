package modeladapter

import (
	"context"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type openAIResponsesAdapter struct{}

// NewOpenAIResponsesAdapter speaks the OpenAI Responses API, which the document
// intelligence path already uses and which Bedrock's mantle endpoint also serves.
func NewOpenAIResponsesAdapter() Adapter { return openAIResponsesAdapter{} }

func (openAIResponsesAdapter) Kind() aiprovider.Kind { return aiprovider.KindOpenAIResponses }

type responsesRequest struct {
	Model           string               `json:"model"`
	Input           []responsesItem      `json:"input"`
	Text            *responsesTextConfig `json:"text,omitempty"`
	Tools           []responsesTool      `json:"tools,omitempty"`
	MaxOutputTokens int                  `json:"max_output_tokens,omitempty"`
}

// responsesItem is both a message and a function call or its output: this
// protocol carries tool traffic as input items rather than as message roles.
type responsesItem struct {
	Type    string                 `json:"type,omitempty"`
	Role    string                 `json:"role,omitempty"`
	Content []responsesMessagePart `json:"content,omitempty"`

	// function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// function_call_output
	Output string `json:"output,omitempty"`
}

type responsesMessagePart struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}

type responsesTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type responsesTextConfig struct {
	Format responsesFormat `json:"format"`
}

type responsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name,omitempty"`
	Schema map[string]any `json:"schema,omitempty"`
	Strict bool           `json:"strict,omitempty"`
}

type responsesEnvelope struct {
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Output []responsesItem `json:"output"`
	Usage  responsesUsage  `json:"usage"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (a openAIResponsesAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := responsesRequest{
		Model:           call.Provider.Model,
		MaxOutputTokens: call.Request.MaxTokens,
		Input:           toResponsesInput(call.Request.System, call.Request.Messages),
		Tools:           toResponsesTools(call.Request.Tools),
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		len(call.Request.Tools) == 0 &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.Text = &responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   call.Request.SchemaName,
				Schema: schema,
				Strict: true,
			},
		}
	}

	var envelope responsesEnvelope
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/responses",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	text, toolCalls, refused := splitResponsesOutput(&envelope)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: firstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.InputTokens,
		OutputTokens:    envelope.Usage.OutputTokens,
		Refused:         refused,
	}, nil
}

func toResponsesInput(system string, messages []Message) []responsesItem {
	items := make([]responsesItem, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		items = append(items, responsesItem{
			Role:    "system",
			Content: []responsesMessagePart{{Type: "input_text", Text: system}},
		})
	}

	for _, msg := range messages {
		switch msg.Role {
		case RoleTool:
			items = append(items, responsesItem{
				Type:   "function_call_output",
				CallID: msg.ToolCallID,
				Output: msg.Content,
			})
		case RoleAssistant:
			if strings.TrimSpace(msg.Content) != "" {
				items = append(items, responsesItem{
					Role:    "assistant",
					Content: []responsesMessagePart{{Type: "output_text", Text: msg.Content}},
				})
			}
			for _, tc := range msg.ToolCalls {
				encoded, err := sonic.Marshal(tc.Arguments)
				if err != nil {
					encoded = []byte("{}")
				}
				items = append(items, responsesItem{
					Type:      "function_call",
					CallID:    tc.ID,
					Name:      tc.Name,
					Arguments: string(encoded),
				})
			}
		default:
			items = append(items, responsesItem{
				Role:    "user",
				Content: []responsesMessagePart{{Type: "input_text", Text: msg.Content}},
			})
		}
	}

	return items
}

func toResponsesTools(tools []ToolSpec) []responsesTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]responsesTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, responsesTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	return out
}

func splitResponsesOutput(resp *responsesEnvelope) (string, []ToolCall, bool) {
	var text string
	toolCalls := make([]ToolCall, 0, len(resp.Output))

	for idx := range resp.Output {
		item := &resp.Output[idx]

		if item.Type == "function_call" {
			args := map[string]any{}
			if trimmed := strings.TrimSpace(item.Arguments); trimmed != "" {
				_ = sonic.Unmarshal([]byte(trimmed), &args)
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: args,
			})
			continue
		}

		for partIdx := range item.Content {
			part := &item.Content[partIdx]
			if strings.TrimSpace(part.Refusal) != "" {
				return "", nil, true
			}
			if text == "" && strings.TrimSpace(part.Text) != "" {
				text = part.Text
			}
		}
	}

	if len(toolCalls) == 0 {
		return text, nil, false
	}

	return text, toolCalls, false
}
