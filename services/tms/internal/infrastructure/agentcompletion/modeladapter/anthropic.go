package modeladapter

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

const anthropicVersion = "2023-06-01"

type anthropicAdapter struct{}

// NewAnthropicAdapter speaks the Anthropic Messages API. Bedrock's mantle
// endpoint serves this same shape, so pointing a provider's base URL there needs
// no separate adapter.
func NewAnthropicAdapter() Adapter { return anthropicAdapter{} }

func (anthropicAdapter) Kind() aiprovider.Kind { return aiprovider.KindAnthropicMessages }

type anthropicRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	System       string                 `json:"system,omitempty"`
	Messages     []anthropicMessage     `json:"messages"`
	Tools        []anthropicTool        `json:"tools,omitempty"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
}

// anthropicMessage carries content as blocks rather than a string, since tool
// use and tool results are block types rather than roles.
type anthropicMessage struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicOutputConfig struct {
	Format anthropicOutputFormat `json:"format"`
}

type anthropicOutputFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type anthropicResponse struct {
	Model      string           `json:"model"`
	StopReason string           `json:"stop_reason"`
	Content    []anthropicBlock `json:"content"`
	Usage      anthropicUsage   `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (a anthropicAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := anthropicRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		System:    call.Request.System,
		Messages:  toAnthropicMessages(call.Request.Messages),
		Tools:     toAnthropicTools(call.Request.Tools),
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		len(call.Request.Tools) == 0 &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.OutputConfig = &anthropicOutputConfig{
			Format: anthropicOutputFormat{Type: "json_schema", Schema: schema},
		}
	}

	var envelope anthropicResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/messages",
		map[string]string{
			"x-api-key":         call.APIKey,
			"anthropic-version": anthropicVersion,
		},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	text, toolCalls := splitAnthropicContent(envelope.Content)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: envelope.Model,
		InputTokens:     envelope.Usage.InputTokens,
		OutputTokens:    envelope.Usage.OutputTokens,
		Refused:         envelope.StopReason == "refusal",
	}, nil
}

func toAnthropicMessages(messages []Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case RoleTool:
			// A tool result is a user-role message carrying a tool_result block,
			// not a role of its own.
			out = append(out, anthropicMessage{
				Role: "user",
				Content: []anthropicBlock{{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   msg.Content,
					IsError:   msg.IsError,
				}},
			})
		case RoleAssistant:
			blocks := make([]anthropicBlock, 0, len(msg.ToolCalls)+1)
			if strings.TrimSpace(msg.Content) != "" {
				blocks = append(blocks, anthropicBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropicBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Arguments,
				})
			}
			if len(blocks) == 0 {
				continue
			}
			out = append(out, anthropicMessage{Role: "assistant", Content: blocks})
		default:
			out = append(out, anthropicMessage{
				Role:    "user",
				Content: []anthropicBlock{{Type: "text", Text: msg.Content}},
			})
		}
	}

	return out
}

func toAnthropicTools(tools []ToolSpec) []anthropicTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	return out
}

func splitAnthropicContent(blocks []anthropicBlock) (string, []ToolCall) {
	var text string
	toolCalls := make([]ToolCall, 0, len(blocks))

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if text == "" && strings.TrimSpace(block.Text) != "" {
				text = block.Text
			}
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	if len(toolCalls) == 0 {
		return text, nil
	}

	return text, toolCalls
}
