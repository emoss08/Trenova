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

const (
	anthropicVersion = "2023-06-01"
	// anthropicBindingBeta opens thinking.block_binding, which is a 400
	// without it.
	anthropicBindingBeta = "thinking-binding-controls-2026-08-01"

	anthropicThinkingAdaptive = "adaptive"
	anthropicFormatJSONSchema = "json_schema"
)

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

// anthropicThinking is how a request asks a model to think. A model from
// before adaptive thinking takes a token budget ("enabled"); a current one
// thinks by effort ("adaptive", the effort in output_config) and refuses a
// budget outright. Display "summarized" asks for the readable summary, which
// the current models leave out unless asked.
type anthropicThinking struct {
	Type         string                 `json:"type"`
	BudgetTokens int                    `json:"budget_tokens,omitempty"`
	Display      string                 `json:"display,omitempty"`
	BlockBinding *anthropicBlockBinding `json:"block_binding,omitempty"`
}

// anthropicBlockBinding is what a model that binds each thinking block to the
// conversation before it does with a block whose conversation changed.
type anthropicBlockBinding struct {
	PrefixMismatchBehavior string `json:"prefix_mismatch_behavior"`
}

// applyThinking asks for thinking the way the configured model takes it.
func (r *anthropicRequest) applyThinking(call *Call) {
	model := providerTraits(call.Provider)
	if !model.adaptive {
		r.applyThinkingBudget(call)
		return
	}

	effort := call.reasoning()
	switch {
	case effort.Enabled():
		r.Thinking = &anthropicThinking{Type: anthropicThinkingAdaptive, Display: "summarized"}
		r.config().Effort = anthropicEffort(effort)
	case effort.Disabled():
		// None is the least thinking the model allows. A model that always
		// thinks, refuses "disabled", or is an effort model the operator named
		// behind an unreadable id, thinks at low effort; one that takes
		// "disabled" is told so; the rest do not think unless asked.
		switch {
		case model.thinksAlways || model.disableRefused || model.declared:
			r.Thinking = &anthropicThinking{Type: anthropicThinkingAdaptive}
			r.config().Effort = "low"
		case model.disableNeedsLowEffort:
			r.Thinking = &anthropicThinking{Type: "disabled"}
		}
	}
	if model.bindsPrefix {
		r.bindThinking()
	}
	if r.Thinking != nil && r.Thinking.Type == anthropicThinkingAdaptive &&
		r.MaxTokens < reasoningAnswerFloor {
		r.MaxTokens = reasoningAnswerFloor
	}
}

// bindThinking asks a model that binds each thinking block to the
// conversation before it to drop a block whose conversation changed, rather
// than refuse the request. Within a turn the replayed history holds still, but
// a tool found mid-turn changes the tool list every earlier block was bound
// to. The binding rides on a thinking object, so Off sends adaptive, which is
// what these models do when told nothing.
func (r *anthropicRequest) bindThinking() {
	if r.Thinking == nil {
		r.Thinking = &anthropicThinking{Type: anthropicThinkingAdaptive}
	}
	r.Thinking.BlockBinding = &anthropicBlockBinding{PrefixMismatchBehavior: "drop_block"}
}

// anthropicHeaders are the headers a request to the configured model carries.
// The binding beta goes only to the models that take it: a gateway in front
// of an older one may refuse a beta it does not know.
func anthropicHeaders(call *Call) map[string]string {
	headers := map[string]string{
		"x-api-key":         call.APIKey,
		"anthropic-version": anthropicVersion,
	}
	if providerTraits(call.Provider).bindsPrefix {
		headers["anthropic-beta"] = anthropicBindingBeta
	}

	return headers
}

// applyThinkingBudget asks a model from before adaptive thinking to think
// with a token budget, and raises the reply ceiling so the budget fits under
// it with room to answer.
func (r *anthropicRequest) applyThinkingBudget(call *Call) {
	budget := call.reasoning().ThinkingBudget()
	if budget == 0 {
		return
	}
	r.Thinking = &anthropicThinking{Type: "enabled", BudgetTokens: budget}
	if floor := budget + thinkingAnswerRoom; r.MaxTokens < floor {
		r.MaxTokens = floor
	}
}

// anthropicEffort is the effort a reasoning level asks a current model for.
// These models have no "minimal"; low is the least.
func anthropicEffort(effort aiprovider.ReasoningEffort) string {
	switch effort {
	case aiprovider.ReasoningMedium:
		return "medium"
	case aiprovider.ReasoningHigh:
		return "high"
	case aiprovider.ReasoningOff,
		aiprovider.ReasoningNone,
		aiprovider.ReasoningMinimal,
		aiprovider.ReasoningLow:
		return "low"
	default:
		return "low"
	}
}

func (r *anthropicRequest) config() *anthropicOutputConfig {
	if r.OutputConfig == nil {
		r.OutputConfig = &anthropicOutputConfig{}
	}

	return r.OutputConfig
}

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
	// Thinking is a pointer because a thinking block must carry the field even
	// when its summary came back empty, and every other block must omit it.
	Thinking  *string `json:"thinking,omitempty"`
	Signature string  `json:"signature,omitempty"`
	Data      string  `json:"data,omitempty"`
	IsError   bool    `json:"is_error,omitempty"`
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
	Format *anthropicOutputFormat `json:"format,omitempty"`
	Effort string                 `json:"effort,omitempty"`
}

type anthropicOutputFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type anthropicResponse struct {
	Model                string                         `json:"model"`
	StopReason           string                         `json:"stop_reason"`
	Content              []anthropicBlock               `json:"content"`
	Usage                anthropicUsage                 `json:"usage"`
	InputTransformations []anthropicInputTransformation `json:"input_transformations"`
}

// anthropicInputTransformation is a change the API made to the request before
// the model read it, such as a thinking block it dropped because the
// conversation before it had changed.
type anthropicInputTransformation struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// thinkingDropped counts the thinking blocks the API dropped from a request.
// An entry that only allowed a mismatched block through is not a drop.
func thinkingDropped(transformations []anthropicInputTransformation) int {
	dropped := 0
	for _, transformation := range transformations {
		if transformation.Type == "thinking_dropped" {
			dropped++
		}
	}

	return dropped
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
		System:    cachedSystem(call.Request.System, call.Request.SystemStable),
		Messages:  cachedConversation(toAnthropicMessages(call.Request.Messages)),
		Tools:     cachedTools(toAnthropicTools(call.Request.Tools)),
	}
	body.applyThinking(call)

	if schema := call.Request.OutputSchema; schema != nil &&
		len(call.Request.Tools) == 0 &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.config().Format = &anthropicOutputFormat{
			Type:   anthropicFormatJSONSchema,
			Schema: schema,
		}
	}

	var envelope anthropicResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/messages",
		anthropicHeaders(call),
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
		OutputLimit:      body.MaxTokens,
		ThinkingDropped:  thinkingDropped(envelope.InputTransformations),
	}, nil
}

// anthropicStreamEvent is the union of every event the Messages API streams.
// Only the fields a given type carries are set; the rest decode to their zero
// values and are ignored.
type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message *struct {
		Model                string                         `json:"model"`
		Usage                anthropicUsage                 `json:"usage"`
		InputTransformations []anthropicInputTransformation `json:"input_transformations"`
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
		System:    cachedSystem(call.Request.System, call.Request.SystemStable),
		Messages:  cachedConversation(toAnthropicMessages(call.Request.Messages)),
		Tools:     cachedTools(toAnthropicTools(call.Request.Tools)),
		Stream:    true,
	}
	body.applyThinking(call)

	stream, err := postStream(
		ctx,
		call,
		call.Provider.ResolvedBaseURL()+"/v1/messages",
		anthropicHeaders(call),
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
		dropped    int
		blocks     = map[int]*anthropicStreamBlock{}
		order      []int
	)

	// messageStopped is the closing event; with the stop reason it is how a
	// finished reply is told from a dropped connection.
	messageStopped := false
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
				dropped = thinkingDropped(event.Message.InputTransformations)
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
		case "message_stop":
			messageStopped = true
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
	if err == nil && stopReason == "" && !messageStopped {
		err = errStreamCut
	}
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
			thinking := streamed.thinking.String()
			block.Thinking = &thinking
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
		OutputLimit:      body.MaxTokens,
		ThinkingDropped:  dropped,
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
			if block.Thinking != nil {
				text.WriteString(*block.Thinking)
			}
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
			Thinking:  &trace.Text,
			Signature: trace.Signature,
		})
	}
	for _, data := range trace.Redacted {
		blocks = append(blocks, anthropicBlock{Type: "redacted_thinking", Data: data})
	}

	return blocks
}

// toAnthropicMessages replays thinking only for the current turn's tool loop,
// the assistant messages after the last user one. A thinking block is bound
// to the conversation before it, and that conversation changes between turns:
// the system prompt carries the turn's page, memories and date, and older tool
// results are shortened. Replaying an earlier turn's block after such an edit
// is a 400 on the models that check, while dropping a leading run of blocks
// is an edit every model accepts, and most ignore earlier turns' thinking
// anyway. A tool result needs its own turn's thinking before it, which stays.
func toAnthropicMessages(messages []Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(messages))
	currentTurn := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			currentTurn = i + 1
			break
		}
	}

	for i, msg := range messages {
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
			if i >= currentTurn {
				blocks = append(blocks, replayThinking(msg.Reasoning)...)
			}
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
			InputSchema: toolschema.ForModel(tool.Parameters),
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

Anthropic matches a cached prefix byte for byte and allows four marks a
request, so they go at the boundaries that are both large and stable: the end
of the tool schemas, the end of the part of the system prompt every turn
shares, and the end of the conversation.

The system prompt leads with what never changes for an agent and ends with the
turn's own context, so its mark sits where the shared part ends and a new page
or memory no longer costs the rules and the tools.

The conversation's mark is what a tool loop lives on. Each call resends the
whole exchange one tool result longer, and the mark on the last block lets the
next call read everything before that result back from the cache. A mark from
an earlier call stays a valid place to read from, so the cache grows with the
conversation rather than being rewritten by it.

The marks go at the end of each block rather than the start, because what is
cached is everything up to the mark. An empty tool list, system prompt or
conversation gets no mark: a breakpoint on nothing still costs a write.
*/
func cachedTools(tools []anthropicTool) []anthropicTool {
	if len(tools) == 0 {
		return tools
	}

	tools[len(tools)-1].CacheControl = ephemeralCache()

	return tools
}

func cachedSystem(system string, stable int) []anthropicBlock {
	if system == "" {
		return nil
	}
	if stable <= 0 || stable >= len(system) {
		return []anthropicBlock{{
			Type:         "text",
			Text:         system,
			CacheControl: ephemeralCache(),
		}}
	}

	return []anthropicBlock{
		{Type: "text", Text: system[:stable], CacheControl: ephemeralCache()},
		{Type: "text", Text: system[stable:]},
	}
}

func cachedConversation(messages []anthropicMessage) []anthropicMessage {
	if len(messages) == 0 {
		return messages
	}
	last := &messages[len(messages)-1]
	if len(last.Content) == 0 {
		return messages
	}
	last.Content[len(last.Content)-1].CacheControl = ephemeralCache()

	return messages
}
