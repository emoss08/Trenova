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
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	// System is sent as blocks rather than a string so the last one can carry a
	// cache breakpoint. Anthropic accepts either shape.
	System       []anthropicBlock       `json:"system,omitempty"`
	Messages     []anthropicMessage     `json:"messages"`
	Tools        []anthropicTool        `json:"tools,omitempty"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
	Stream       bool                   `json:"stream,omitempty"`
	Thinking     *anthropicThinking     `json:"thinking,omitempty"`
}

// anthropicThinking turns extended thinking on with a token budget. The
// budget must be below max_tokens, which the request builder guarantees.
type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// applyThinking asks for extended thinking at the provider's effort and
// raises the reply ceiling so the budget fits under it with room to answer.
func (r *anthropicRequest) applyThinking(call *Call) {
	budget := call.reasoning().ThinkingBudget()
	if budget == 0 {
		return
	}
	r.Thinking = &anthropicThinking{Type: "enabled", BudgetTokens: budget}
	if floor := budget + thinkingAnswerRoom; r.MaxTokens < floor {
		r.MaxTokens = floor
	}
}

// thinkingAnswerRoom is what the answer keeps after the thinking budget.
const thinkingAnswerRoom = 2048

// anthropicMessage carries content as blocks rather than a string, since tool
// use and tool results are block types rather than roles.
type anthropicMessage struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`

	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
	// InputError is why streamed input JSON did not parse; never on the wire.
	InputError string `json:"-"`

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`

	// thinking and redacted_thinking. The signature is the provider's proof
	// that the block is its own; a tool result replayed without it is refused.
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]any         `json:"input_schema"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

// anthropicCacheControl marks the end of a prefix worth keeping. Everything
// before the mark is cached; a later request whose bytes match up to that point
// reads it back instead of paying for it again.
type anthropicCacheControl struct {
	Type string `json:"type"`
}

func ephemeralCache() *anthropicCacheControl {
	return &anthropicCacheControl{Type: "ephemeral"}
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
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func (u *anthropicUsage) merge(update *anthropicUsage) {
	if update == nil {
		return
	}
	if update.InputTokens > 0 {
		u.InputTokens = update.InputTokens
	}
	if update.OutputTokens > 0 {
		u.OutputTokens = update.OutputTokens
	}
	if update.CacheReadInputTokens > 0 {
		u.CacheReadInputTokens = update.CacheReadInputTokens
	}
	if update.CacheCreationInputTokens > 0 {
		u.CacheCreationInputTokens = update.CacheCreationInputTokens
	}
}

func (a anthropicAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := anthropicRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		System:    cachedSystem(call.Request.System),
		Messages:  toAnthropicMessages(call.Request.Messages),
		Tools:     cachedTools(toAnthropicTools(call.Request.Tools)),
	}
	body.applyThinking(call)

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
		Text:             text,
		ToolCalls:        toolCalls,
		ModelIdentifier:  envelope.Model,
		InputTokens:      envelope.Usage.InputTokens,
		OutputTokens:     envelope.Usage.OutputTokens,
		CacheReadTokens:  envelope.Usage.CacheReadInputTokens,
		CacheWriteTokens: envelope.Usage.CacheCreationInputTokens,
		Refused:          envelope.StopReason == "refusal",
		Truncated:        envelope.StopReason == "max_tokens",
		Reasoning:        anthropicReasoning(envelope.Content),
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
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
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
	block    anthropicBlock
	text     strings.Builder
	input    strings.Builder
	thinking strings.Builder
}

func (a anthropicAdapter) Stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
) (*Response, error) {
	body := anthropicRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		System:    cachedSystem(call.Request.System),
		Messages:  toAnthropicMessages(call.Request.Messages),
		Tools:     cachedTools(toAnthropicTools(call.Request.Tools)),
		Stream:    true,
	}
	body.applyThinking(call)

	stream, err := postStream(
		ctx,
		call,
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
				usage.merge(&event.Message.Usage)
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
			case "thinking_delta":
				block.thinking.WriteString(event.Delta.Thinking)
				call.think(event.Delta.Thinking)
			case "signature_delta":
				block.block.Signature += event.Delta.Signature
			}
		case "message_delta":
			if event.Delta != nil {
				stopReason = stringutils.FirstNonEmpty(event.Delta.StopReason, stopReason)
			}
			usage.merge(event.Usage)
		case "error":
			if event.Error != nil {
				return streamError(event.Error.Type, event.Error.Message)
			}
			return streamError("", "")
		}

		return nil
	})
	if err != nil {
		return nil, interrupted(err, model)
	}

	content := make([]anthropicBlock, 0, len(order))
	for _, index := range order {
		streamed := blocks[index]
		block := streamed.block
		switch block.Type {
		case "text":
			block.Text = streamed.text.String()
		case "thinking":
			block.Thinking = streamed.thinking.String()
		case "tool_use":
			if raw := streamed.input.String(); strings.TrimSpace(raw) != "" {
				block.Input, block.InputError = decodeArguments(raw)
			}
			if block.Input == nil {
				block.Input = map[string]any{}
			}
		}
		content = append(content, block)
	}

	text, toolCalls := splitAnthropicContent(content)

	return &Response{
		Text:             text,
		ToolCalls:        toolCalls,
		ModelIdentifier:  stringutils.FirstNonEmpty(model, call.Provider.Model),
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CacheReadTokens:  usage.CacheReadInputTokens,
		CacheWriteTokens: usage.CacheCreationInputTokens,
		Refused:          stopReason == "refusal",
		Truncated:        stopReason == "max_tokens",
		Reasoning:        anthropicReasoning(content),
	}, nil
}

// anthropicReasoning gathers the thinking blocks of a reply into one trace.
// Redacted blocks carry nothing readable and are kept only to be replayed.
func anthropicReasoning(blocks []anthropicBlock) *ReasoningTrace {
	var trace ReasoningTrace
	var text strings.Builder
	found := false
	for _, block := range blocks {
		switch block.Type {
		case "thinking":
			found = true
			text.WriteString(block.Thinking)
			trace.Signature = stringutils.FirstNonEmpty(trace.Signature, block.Signature)
		case "redacted_thinking":
			found = true
			trace.Redacted = append(trace.Redacted, block.Data)
		}
	}
	if !found {
		return nil
	}
	trace.Text = text.String()

	return &trace
}

// replayThinking is the thinking a previous assistant turn must open with.
// Anthropic refuses a tool result whose preceding thinking is missing, so the
// signed block goes back exactly as it came, redacted blocks included.
func replayThinking(trace *ReasoningTrace) []anthropicBlock {
	if !trace.ReplayableBy(string(aiprovider.KindAnthropicMessages)) {
		return nil
	}
	blocks := make([]anthropicBlock, 0, len(trace.Redacted)+1)
	if trace.Signature != "" {
		blocks = append(blocks, anthropicBlock{
			Type:      "thinking",
			Thinking:  trace.Text,
			Signature: trace.Signature,
		})
	}
	for _, data := range trace.Redacted {
		blocks = append(blocks, anthropicBlock{Type: "redacted_thinking", Data: data})
	}

	return blocks
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
			blocks := make([]anthropicBlock, 0, len(msg.ToolCalls)+2)
			blocks = append(blocks, replayThinking(msg.Reasoning)...)
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
				ID:             block.ID,
				Name:           block.Name,
				Arguments:      block.Input,
				ArgumentsError: block.InputError,
			})
		}
	}

	if len(toolCalls) == 0 {
		return text, nil
	}

	return text, toolCalls
}

/*
Where the prefix is worth keeping.

Anthropic matches a cached prefix byte for byte and allows a handful of marks,
so they go at the two boundaries that are both large and unchanging: the end of
the tool schemas and the end of the system prompt. Those two are most of what a
turn sends and every iteration of a tool loop resends them verbatim — the
second call in a two-tool turn re-read the whole prompt and every schema before
this.

The marks go at the end of each block rather than the start, because what is
cached is everything up to the mark. Nothing marks the conversation itself: it
grows every turn, so a mark there caches a prefix that the next request has
already moved past.

An empty tool list or system prompt gets no mark. A breakpoint on nothing still
costs a write.
*/
func cachedTools(tools []anthropicTool) []anthropicTool {
	if len(tools) == 0 {
		return tools
	}

	tools[len(tools)-1].CacheControl = ephemeralCache()

	return tools
}

func cachedSystem(system string) []anthropicBlock {
	if system == "" {
		return nil
	}

	return []anthropicBlock{{
		Type:         "text",
		Text:         system,
		CacheControl: ephemeralCache(),
	}}
}
