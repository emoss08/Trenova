package modeladapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/stringutils"
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
	Stream       bool                   `json:"stream,omitempty"`
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

// anthropicStreamEvent is the union of every event the Messages API streams.
// Only the fields a given type carries are set; the rest decode to their zero
// values and are ignored.
type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message *struct {
		Model string         `json:"model"`
		Usage anthropicUsage `json:"usage"`
	} `json:"message"`
	ContentBlock *anthropicBlock `json:"content_block"`
	Delta        *struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	Usage *anthropicUsage `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// anthropicStreamBlock accumulates one content block as its deltas arrive. Text
// grows by text_delta; a tool_use block's input arrives as fragments of one
// JSON document that is only parseable once the block stops.
type anthropicStreamBlock struct {
	block anthropicBlock
	text  strings.Builder
	input strings.Builder
}

func (a anthropicAdapter) Stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
) (*Response, error) {
	body := anthropicRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		System:    call.Request.System,
		Messages:  toAnthropicMessages(call.Request.Messages),
		Tools:     toAnthropicTools(call.Request.Tools),
		Stream:    true,
	}

	stream, err := postStream(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/messages",
		map[string]string{
			"x-api-key":         call.APIKey,
			"anthropic-version": anthropicVersion,
		},
		body,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var (
		model      string
		usage      anthropicUsage
		stopReason string
		blocks     = map[int]*anthropicStreamBlock{}
		order      []int
	)

	err = readSSE(stream, func(_, data string) error {
		var event anthropicStreamEvent
		if err := sonic.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode stream event: %w", err)
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				model = event.Message.Model
				usage.InputTokens = event.Message.Usage.InputTokens
			}
		case "content_block_start":
			if event.ContentBlock == nil {
				return nil
			}
			block := &anthropicStreamBlock{block: *event.ContentBlock}
			block.text.WriteString(event.ContentBlock.Text)
			blocks[event.Index] = block
			order = append(order, event.Index)
		case "content_block_delta":
			block, ok := blocks[event.Index]
			if !ok || event.Delta == nil {
				return nil
			}
			switch event.Delta.Type {
			case "text_delta":
				block.text.WriteString(event.Delta.Text)
				if event.Delta.Text != "" {
					sink(event.Delta.Text)
				}
			case "input_json_delta":
				block.input.WriteString(event.Delta.PartialJSON)
			}
		case "message_delta":
			if event.Delta != nil {
				stopReason = stringutils.FirstNonEmpty(event.Delta.StopReason, stopReason)
			}
			if event.Usage != nil {
				usage.OutputTokens = event.Usage.OutputTokens
			}
		case "error":
			if event.Error != nil {
				return streamError(event.Error.Type, event.Error.Message)
			}
			return streamError("", "")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	content := make([]anthropicBlock, 0, len(order))
	for _, index := range order {
		streamed := blocks[index]
		block := streamed.block
		switch block.Type {
		case "text":
			block.Text = streamed.text.String()
		case "tool_use":
			if raw := streamed.input.String(); strings.TrimSpace(raw) != "" {
				block.Input = decodeArguments(raw)
			}
			if block.Input == nil {
				block.Input = map[string]any{}
			}
		}
		content = append(content, block)
	}

	text, toolCalls := splitAnthropicContent(content)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: stringutils.FirstNonEmpty(model, call.Provider.Model),
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		Refused:         stopReason == "refusal",
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
