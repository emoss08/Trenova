package modeladapter

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

// anthropicBindingBeta opens thinking.block_binding, which is a 400 without
// it.
const anthropicBindingBeta = anthropic.AnthropicBetaThinkingBindingControls2026_08_01

const (
	anthropicTextBlock     = "text"
	anthropicToolUseBlock  = "tool_use"
	anthropicThinkingBlock = "thinking"
	anthropicRedactedBlock = "redacted_thinking"
)

type anthropicAdapter struct {
	credentials *platformCredentials
}

// NewAnthropicAdapter speaks the Anthropic Messages API, called directly or
// through the platform a provider names.
func NewAnthropicAdapter() Adapter {
	return anthropicAdapter{credentials: newPlatformCredentials()}
}

func (anthropicAdapter) Kind() aiprovider.Kind { return aiprovider.KindAnthropicMessages }

// anthropicParams is the request every call sends: the tools, the system
// prompt and the conversation, each marked where its prefix is worth keeping,
// and thinking asked for the way the configured model takes it.
func anthropicParams(call *Call, attached *anthropicFiles) anthropic.BetaMessageNewParams {
	params := anthropic.BetaMessageNewParams{
		Model:     call.Provider.Model,
		MaxTokens: int64(call.Request.MaxTokens),
		System:    cachedSystem(call.Request.System, call.Request.SystemStable),
		Messages: cachedConversation(
			anthropicConversation(attached.messages, attached.blocks),
		),
		Tools: cachedTools(toAnthropicTools(call.Request.Tools)),
	}
	if call.Request.ToolChoiceNone && len(params.Tools) > 0 {
		none := anthropic.NewBetaToolChoiceNoneParam()
		params.ToolChoice = anthropic.BetaToolChoiceUnionParam{OfNone: &none}
	}
	applyThinking(&params, call)
	applyDiagnostics(&params, call)

	return params
}

// applyThinking asks for thinking the way the configured model takes it.
func applyThinking(params *anthropic.BetaMessageNewParams, call *Call) {
	model := providerTraits(call.Provider)
	if !model.adaptive {
		applyThinkingBudget(params, call)
		return
	}

	effort := call.reasoning()
	switch {
	case effort.Enabled():
		params.Thinking = adaptiveThinking(anthropic.BetaThinkingConfigAdaptiveDisplaySummarized)
		params.OutputConfig.Effort = anthropicEffort(effort)
	case effort.Disabled():
		// None is the least thinking the model allows. A model that always
		// thinks, refuses "disabled", or is an effort model the operator named
		// behind an unreadable id, thinks at low effort; one that takes
		// "disabled" is told so; the rest do not think unless asked.
		switch {
		case model.thinksAlways || model.disableRefused || model.declared:
			params.Thinking = adaptiveThinking("")
			params.OutputConfig.Effort = anthropic.BetaOutputConfigEffortLow
		case model.disableNeedsLowEffort:
			params.Thinking = anthropic.BetaThinkingConfigParamUnion{
				OfDisabled: &anthropic.BetaThinkingConfigDisabledParam{},
			}
		}
	}
	if model.bindsPrefix {
		bindThinking(params)
	}
	if params.Thinking.OfAdaptive != nil && params.MaxTokens < reasoningAnswerFloor {
		params.MaxTokens = reasoningAnswerFloor
	}
}

func adaptiveThinking(
	display anthropic.BetaThinkingConfigAdaptiveDisplay,
) anthropic.BetaThinkingConfigParamUnion {
	return anthropic.BetaThinkingConfigParamUnion{
		OfAdaptive: &anthropic.BetaThinkingConfigAdaptiveParam{Display: display},
	}
}

// bindThinking asks a model that binds each thinking block to the
// conversation before it to drop a block whose conversation changed, rather
// than refuse the request. Within a turn the replayed history holds still, but
// a tool found mid-turn changes the tool list every earlier block was bound
// to. The binding rides on a thinking object, so Off sends adaptive, which is
// what these models do when told nothing. The binding beta goes only to the
// models that take it: a gateway in front of an older one may refuse a beta it
// does not know.
func bindThinking(params *anthropic.BetaMessageNewParams) {
	binding := anthropic.BetaThinkingBlockBindingParam{
		PrefixMismatchBehavior: anthropic.BetaThinkingPrefixMismatchBehaviorDropBlock,
	}
	switch {
	case params.Thinking.OfAdaptive != nil:
		params.Thinking.OfAdaptive.BlockBinding = binding
	case params.Thinking.OfEnabled != nil:
		params.Thinking.OfEnabled.BlockBinding = binding
	case params.Thinking.OfDisabled == nil:
		params.Thinking = anthropic.BetaThinkingConfigParamUnion{
			OfAdaptive: &anthropic.BetaThinkingConfigAdaptiveParam{BlockBinding: binding},
		}
	}
	params.Betas = append(params.Betas, anthropicBindingBeta)
}

// applyThinkingBudget asks a model from before adaptive thinking to think
// with a token budget, and raises the reply ceiling so the budget fits under
// it with room to answer.
func applyThinkingBudget(params *anthropic.BetaMessageNewParams, call *Call) {
	budget := call.reasoning().ThinkingBudget()
	if budget == 0 {
		return
	}
	params.Thinking = anthropic.BetaThinkingConfigParamUnion{
		OfEnabled: &anthropic.BetaThinkingConfigEnabledParam{BudgetTokens: int64(budget)},
	}
	if floor := int64(budget + thinkingAnswerRoom); params.MaxTokens < floor {
		params.MaxTokens = floor
	}
}

// anthropicEffort is the effort a reasoning level asks a current model for.
// These models have no "minimal"; low is the least.
func anthropicEffort(effort aiprovider.ReasoningEffort) anthropic.BetaOutputConfigEffort {
	switch effort {
	case aiprovider.ReasoningMedium:
		return anthropic.BetaOutputConfigEffortMedium
	case aiprovider.ReasoningHigh:
		return anthropic.BetaOutputConfigEffortHigh
	case aiprovider.ReasoningOff,
		aiprovider.ReasoningNone,
		aiprovider.ReasoningMinimal,
		aiprovider.ReasoningLow:
		return anthropic.BetaOutputConfigEffortLow
	default:
		return anthropic.BetaOutputConfigEffortLow
	}
}

// applyOutputSchema asks for a reply that satisfies the schema, where the
// provider enforces one and no tools are offered alongside it.
func applyOutputSchema(params *anthropic.BetaMessageNewParams, call *Call) {
	schema := call.Request.OutputSchema
	if schema == nil || len(call.Request.Tools) > 0 ||
		call.Provider.StructuredOutputMode != aiprovider.StructuredOutputJSONSchema {
		return
	}
	params.OutputConfig.Format = anthropic.BetaJSONOutputFormatParam{
		Schema: anthropicOutputSchema(schema),
	}
}

func (a anthropicAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	attached := attachFiles(ctx, call)
	resp, err := a.complete(ctx, call, &attached)
	if refusedOverFiles(err, &attached) {
		plain := plainConversation(call, &attached)
		resp, err = a.complete(ctx, call, &plain)
	}

	return resp, err
}

func (a anthropicAdapter) complete(
	ctx context.Context,
	call *Call,
	attached *anthropicFiles,
) (*Response, error) {
	params := anthropicParams(call, attached)
	applyOutputSchema(&params, call)

	service, err := anthropicMessages(ctx, call, false, a.credentials)
	if err != nil {
		return nil, err
	}
	message, err := service.New(ctx, params, anthropicBlockingOptions(call)...)
	if err != nil {
		return nil, err
	}

	content := anthropicContent(message.Content)
	text, toolCalls := splitAnthropicContent(content)

	return &Response{
		Text:             text,
		ToolCalls:        toolCalls,
		ModelIdentifier:  message.Model,
		InputTokens:      int(message.Usage.InputTokens),
		OutputTokens:     int(message.Usage.OutputTokens),
		CacheReadTokens:  int(message.Usage.CacheReadInputTokens),
		CacheWriteTokens: int(message.Usage.CacheCreationInputTokens),
		Refused:          message.StopReason == anthropic.BetaStopReasonRefusal,
		Truncated:        message.StopReason == anthropic.BetaStopReasonMaxTokens,
		Reasoning:        anthropicReasoning(content),
		OutputLimit:      int(params.MaxTokens),
		ThinkingDropped:  thinkingDropped(message.InputTransformations),
		MessageID:        message.ID,
		CacheDiagnosis:   anthropicDiagnosis(message),
		Files:            attached.report,
	}, nil
}

// anthropicBlock is one block of a reply, read off the SDK's union into the
// fields the adapter uses.
type anthropicBlock struct {
	Type      string
	Text      string
	ID        string
	Name      string
	Arguments map[string]any
	// InputError is why a tool call's input did not parse.
	InputError string
	Thinking   string
	Signature  string
	Data       string
}

func anthropicContent(blocks []anthropic.BetaContentBlockUnion) []anthropicBlock {
	content := make([]anthropicBlock, 0, len(blocks))
	for idx := range blocks {
		block := &blocks[idx]
		read := anthropicBlock{
			Type:      block.Type,
			Text:      block.Text,
			ID:        block.ID,
			Name:      block.Name,
			Thinking:  block.Thinking,
			Signature: block.Signature,
			Data:      block.Data,
		}
		if block.Type == anthropicToolUseBlock {
			read.Arguments, read.InputError = decodeArguments(string(block.Input))
		}
		content = append(content, read)
	}

	return content
}

// thinkingDropped counts the thinking blocks the API dropped from a request.
// An entry that only allowed a mismatched block through is not a drop.
func thinkingDropped(transformations []anthropic.BetaInputTransformationUnion) int {
	dropped := 0
	for idx := range transformations {
		if transformations[idx].Type == "thinking_dropped" {
			dropped++
		}
	}

	return dropped
}

type anthropicUsage struct {
	InputTokens              int64
	OutputTokens             int64
	CacheReadInputTokens     int64
	CacheCreationInputTokens int64
}

func (u *anthropicUsage) merge(update anthropicUsage) {
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
	attached := attachFiles(ctx, call)
	emitted := false
	watched := func(text string) {
		emitted = true
		sink(text)
	}
	resp, err := a.stream(ctx, call, watched, &attached)
	if !emitted && refusedOverFiles(err, &attached) {
		plain := plainConversation(call, &attached)
		resp, err = a.stream(ctx, call, sink, &plain)
	}

	return resp, err
}

func (a anthropicAdapter) stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
	attached *anthropicFiles,
) (*Response, error) {
	params := anthropicParams(call, attached)

	service, err := anthropicMessages(ctx, call, true, a.credentials)
	if err != nil {
		return nil, err
	}
	stream := service.NewStreaming(ctx, params)
	defer func() { _ = stream.Close() }()

	var (
		model      string
		usage      anthropicUsage
		stopReason string
		dropped    int
		messageID  string
		diagnosis  *CacheDiagnosis
		blocks     = map[int64]*anthropicStreamBlock{}
		order      []int64
	)

	// messageStopped is the closing event; with the stop reason it is how a
	// finished reply is told from a dropped connection.
	messageStopped := false
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "message_start":
			model = event.Message.Model
			usage.merge(anthropicUsage{
				InputTokens:              event.Message.Usage.InputTokens,
				OutputTokens:             event.Message.Usage.OutputTokens,
				CacheReadInputTokens:     event.Message.Usage.CacheReadInputTokens,
				CacheCreationInputTokens: event.Message.Usage.CacheCreationInputTokens,
			})
			dropped = thinkingDropped(event.Message.InputTransformations)
			messageID = event.Message.ID
			diagnosis = anthropicDiagnosis(&event.Message)
		case "content_block_start":
			blocks[event.Index] = openedBlock(&event.ContentBlock)
			order = append(order, event.Index)
		case "content_block_delta":
			block, ok := blocks[event.Index]
			if !ok {
				continue
			}
			delta := event.Delta
			switch delta.Type {
			case "text_delta":
				block.text.WriteString(delta.Text)
				if delta.Text != "" {
					sink(delta.Text)
				}
			case "input_json_delta":
				block.input.WriteString(delta.PartialJSON)
			case "thinking_delta":
				block.thinking.WriteString(delta.Thinking)
				call.think(delta.Thinking)
			case "signature_delta":
				block.block.Signature += delta.Signature
			}
		case "message_stop":
			messageStopped = true
		case "message_delta":
			stopReason = stringutils.FirstNonEmpty(string(event.Delta.StopReason), stopReason)
			usage.merge(anthropicUsage{
				InputTokens:              event.Usage.InputTokens,
				OutputTokens:             event.Usage.OutputTokens,
				CacheReadInputTokens:     event.Usage.CacheReadInputTokens,
				CacheCreationInputTokens: event.Usage.CacheCreationInputTokens,
			})
		}
	}
	err = anthropicStreamFailure(stream.Err())
	if err == nil && stopReason == "" && !messageStopped {
		err = errStreamCut
	}
	if err != nil {
		return nil, interrupted(err, model)
	}

	content := streamedContent(blocks, order)
	text, toolCalls := splitAnthropicContent(content)

	return &Response{
		Text:             text,
		ToolCalls:        toolCalls,
		ModelIdentifier:  stringutils.FirstNonEmpty(model, call.Provider.Model),
		InputTokens:      int(usage.InputTokens),
		OutputTokens:     int(usage.OutputTokens),
		CacheReadTokens:  int(usage.CacheReadInputTokens),
		CacheWriteTokens: int(usage.CacheCreationInputTokens),
		Refused:          stopReason == string(anthropic.BetaStopReasonRefusal),
		Truncated:        stopReason == string(anthropic.BetaStopReasonMaxTokens),
		Reasoning:        anthropicReasoning(content),
		OutputLimit:      int(params.MaxTokens),
		ThinkingDropped:  dropped,
		MessageID:        messageID,
		CacheDiagnosis:   diagnosis,
		Files:            attached.report,
	}, nil
}

// openedBlock is a block as the stream opens it, before any delta.
func openedBlock(
	start *anthropic.BetaRawContentBlockStartEventContentBlockUnion,
) *anthropicStreamBlock {
	block := &anthropicStreamBlock{block: anthropicBlock{
		Type:      start.Type,
		ID:        start.ID,
		Name:      start.Name,
		Signature: start.Signature,
		Data:      start.Data,
	}}
	block.text.WriteString(start.Text)

	return block
}

// streamedContent is the reply's blocks in the order they opened, each with
// the text, thinking or arguments its deltas added up to.
func streamedContent(blocks map[int64]*anthropicStreamBlock, order []int64) []anthropicBlock {
	content := make([]anthropicBlock, 0, len(order))
	for _, index := range order {
		streamed := blocks[index]
		block := streamed.block
		switch block.Type {
		case anthropicTextBlock:
			block.Text = streamed.text.String()
		case anthropicThinkingBlock:
			block.Thinking = streamed.thinking.String()
		case anthropicToolUseBlock:
			block.Arguments, block.InputError = decodeArguments(streamed.input.String())
		}
		content = append(content, block)
	}

	return content
}

// anthropicStreamFailure reads an error event the provider sent inside an
// otherwise healthy stream as the failure it names, retryable when the fault
// is on the provider's side. Anything else is returned as it came.
func anthropicStreamFailure(err error) error {
	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) {
		return err
	}

	var envelope errorEnvelope
	_ = sonic.UnmarshalString(apiErr.RawJSON(), &envelope)

	return streamError(envelope.Error.Type, envelope.Error.Message)
}

// anthropicReasoning gathers the thinking blocks of a reply into one trace.
// Redacted blocks carry nothing readable and are kept only to be replayed.
func anthropicReasoning(blocks []anthropicBlock) *ReasoningTrace {
	var trace ReasoningTrace
	var text strings.Builder
	found := false
	for idx := range blocks {
		block := &blocks[idx]
		switch block.Type {
		case anthropicThinkingBlock:
			found = true
			text.WriteString(block.Thinking)
			trace.Signature = stringutils.FirstNonEmpty(trace.Signature, block.Signature)
		case anthropicRedactedBlock:
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
func replayThinking(trace *ReasoningTrace) []anthropic.BetaContentBlockParamUnion {
	if !trace.ReplayableBy(string(aiprovider.KindAnthropicMessages)) {
		return nil
	}
	blocks := make([]anthropic.BetaContentBlockParamUnion, 0, len(trace.Redacted)+1)
	if trace.Signature != "" {
		blocks = append(blocks, anthropic.BetaContentBlockParamUnion{
			OfThinking: &anthropic.BetaThinkingBlockParam{
				Thinking:  trace.Text,
				Signature: trace.Signature,
			},
		})
	}
	for _, data := range trace.Redacted {
		blocks = append(blocks, anthropic.BetaContentBlockParamUnion{
			OfRedactedThinking: &anthropic.BetaRedactedThinkingBlockParam{Data: data},
		})
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
func toAnthropicMessages(messages []Message) []anthropic.BetaMessageParam {
	return anthropicConversation(messages, nil)
}

// anthropicConversation is the conversation as the API takes it, with the
// blocks that carry each user message's files ahead of its text.
func anthropicConversation(
	messages []Message,
	files map[int][]anthropic.BetaContentBlockParamUnion,
) []anthropic.BetaMessageParam {
	out := make([]anthropic.BetaMessageParam, 0, len(messages))
	currentTurn := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			currentTurn = i + 1
			break
		}
	}

	for i := range messages {
		msg := &messages[i]
		switch msg.Role {
		case RoleTool:
			// A tool result is a tool_result block in a user-role message, not
			// a role of its own. The results of one turn's calls share a
			// message: sent one per message, they teach the model to stop
			// calling tools in parallel.
			block := anthropic.BetaContentBlockParamUnion{OfToolResult: anthropicToolResult(msg)}
			if last, results := lastToolResults(out); last != nil {
				last.Content = slices.Insert(last.Content, results, block)
				continue
			}
			out = append(out, anthropic.BetaMessageParam{
				Role:    anthropic.BetaMessageParamRoleUser,
				Content: []anthropic.BetaContentBlockParamUnion{block},
			})
		case RoleAssistant:
			blocks := make([]anthropic.BetaContentBlockParamUnion, 0, len(msg.ToolCalls)+2)
			if i >= currentTurn {
				blocks = append(blocks, replayThinking(msg.Reasoning)...)
			}
			if strings.TrimSpace(msg.Content) != "" {
				blocks = append(blocks, anthropic.BetaContentBlockParamUnion{
					OfText: &anthropic.BetaTextBlockParam{Text: msg.Content},
				})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropic.BetaContentBlockParamUnion{
					OfToolUse: &anthropic.BetaToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: toolUseInput(tc.Arguments),
					},
				})
			}
			if len(blocks) == 0 {
				continue
			}
			out = append(out, anthropic.BetaMessageParam{
				Role:    anthropic.BetaMessageParamRoleAssistant,
				Content: blocks,
			})
		default:
			blocks := append(slices.Clone(files[i]), anthropic.BetaContentBlockParamUnion{
				OfText: &anthropic.BetaTextBlockParam{Text: msg.Content},
			})
			// Text that follows tool results joins their message, after them,
			// which is the order the API asks for.
			if last, _ := lastToolResults(out); last != nil {
				last.Content = append(last.Content, blocks...)
				continue
			}
			out = append(out, anthropic.BetaMessageParam{
				Role:    anthropic.BetaMessageParamRoleUser,
				Content: blocks,
			})
		}
	}

	return out
}

// lastToolResults is the final message when it is a user message that opens
// with tool results, and how many it opens with. A further result goes in after
// them and any text after it, since the API takes every result ahead of any
// text. A person's own message never opens with a result, so it is never the
// message returned.
func lastToolResults(
	out []anthropic.BetaMessageParam,
) (message *anthropic.BetaMessageParam, results int) {
	if len(out) == 0 {
		return nil, 0
	}
	last := &out[len(out)-1]
	if last.Role != anthropic.BetaMessageParamRoleUser {
		return nil, 0
	}
	for results < len(last.Content) && last.Content[results].OfToolResult != nil {
		results++
	}
	if results == 0 {
		return nil, 0
	}

	return last, results
}

// anthropicToolResult carries a tool's output as a text block. An empty
// output sends no content, since an empty text block is refused.
func anthropicToolResult(msg *Message) *anthropic.BetaToolResultBlockParam {
	result := &anthropic.BetaToolResultBlockParam{ToolUseID: msg.ToolCallID}
	if msg.Content != "" {
		result.Content = []anthropic.BetaToolResultBlockParamContentUnion{{
			OfText: &anthropic.BetaTextBlockParam{Text: msg.Content},
		}}
	}
	if msg.IsError {
		result.IsError = param.NewOpt(true)
	}

	return result
}

func toAnthropicTools(tools []ToolSpec) []anthropic.BetaToolUnionParam {
	if len(tools) == 0 {
		return nil
	}

	out := make([]anthropic.BetaToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		out = append(out, anthropic.BetaToolUnionParam{
			OfTool: &anthropic.BetaToolParam{
				Name:        tool.Name,
				Description: param.NewOpt(tool.Description),
				InputSchema: param.Override[anthropic.BetaToolInputSchemaParam](
					toolschema.ForModel(tool.Parameters),
				),
			},
		})
	}

	return out
}

func toolUseInput(arguments map[string]any) map[string]any {
	if arguments == nil {
		return map[string]any{}
	}

	return arguments
}

func splitAnthropicContent(blocks []anthropicBlock) (string, []ToolCall) {
	var text string
	toolCalls := make([]ToolCall, 0, len(blocks))

	for idx := range blocks {
		block := &blocks[idx]
		switch block.Type {
		case anthropicTextBlock:
			if text == "" && strings.TrimSpace(block.Text) != "" {
				text = block.Text
			}
		case anthropicToolUseBlock:
			toolCalls = append(toolCalls, ToolCall{
				ID:             block.ID,
				Name:           block.Name,
				Arguments:      toolUseInput(block.Arguments),
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
func cachedTools(tools []anthropic.BetaToolUnionParam) []anthropic.BetaToolUnionParam {
	if len(tools) == 0 {
		return tools
	}
	if mark := tools[len(tools)-1].GetCacheControl(); mark != nil {
		*mark = anthropic.NewBetaCacheControlEphemeralParam()
	}

	return tools
}

func cachedSystem(system string, stable int) []anthropic.BetaTextBlockParam {
	if system == "" {
		return nil
	}
	if stable <= 0 || stable >= len(system) {
		return []anthropic.BetaTextBlockParam{{
			Text:         system,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}}
	}

	return []anthropic.BetaTextBlockParam{
		{Text: system[:stable], CacheControl: anthropic.NewBetaCacheControlEphemeralParam()},
		{Text: system[stable:]},
	}
}

func cachedConversation(messages []anthropic.BetaMessageParam) []anthropic.BetaMessageParam {
	if len(messages) == 0 {
		return messages
	}
	last := &messages[len(messages)-1]
	if len(last.Content) == 0 {
		return messages
	}
	if mark := last.Content[len(last.Content)-1].GetCacheControl(); mark != nil {
		*mark = anthropic.NewBetaCacheControlEphemeralParam()
	}

	return messages
}
