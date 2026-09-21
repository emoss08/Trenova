package modeladapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/stringutils"
)

type openAIResponsesAdapter struct{}

var _ BackgroundRunner = openAIResponsesAdapter{}

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
	Stream          bool                 `json:"stream,omitempty"`
	// Reasoning is sent only when the provider is configured to reason. The
	// summary is what a person gets to read; the chain itself never leaves
	// OpenAI in the clear, only encrypted, and only when asked for by Include.
	Reasoning  *responsesReasoning `json:"reasoning,omitempty"`
	Include    []string            `json:"include,omitempty"`
	Background bool                `json:"background,omitempty"`
	Store      bool                `json:"store,omitempty"`
}

// responsesItem is both a message and a function call or its output: this
// protocol carries tool traffic as input items rather than as message roles.
type responsesReasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

// applyReasoning asks for reasoning at the provider's effort, with a summary
// to show and the encrypted chain to replay on the next call.
func (r *responsesRequest) applyReasoning(call *Call) {
	effort := call.reasoning().Wire()
	if effort == "" {
		return
	}
	r.Reasoning = &responsesReasoning{Effort: effort, Summary: "auto"}
	r.Include = []string{"reasoning.encrypted_content"}
}

// responsesSummaryPart is one readable piece of a reasoning item's summary.
type responsesSummaryPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesItem struct {
	Type    string                 `json:"type,omitempty"`
	Role    string                 `json:"role,omitempty"`
	Content []responsesMessagePart `json:"content,omitempty"`

	// reasoning. A function call is refused when replayed without the
	// reasoning item that produced it, so the id and encrypted content go
	// back ahead of the calls exactly as they came.
	ID               string                 `json:"id,omitempty"`
	Summary          []responsesSummaryPart `json:"summary,omitempty"`
	EncryptedContent string                 `json:"encrypted_content,omitempty"`

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
	ID     string          `json:"id"`
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Output []responsesItem `json:"output"`
	Usage  responsesUsage  `json:"usage"`

	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type responsesUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func (a openAIResponsesAdapter) requestFor(call *Call) responsesRequest {
	body := responsesRequest{
		Model:           call.Provider.Model,
		MaxOutputTokens: call.Request.MaxTokens,
		Input:           toResponsesInput(call.Request.System, call.Request.Messages),
		Tools:           toResponsesTools(call.Request.Tools),
	}
	body.applyReasoning(call)

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

	return body
}

func (a openAIResponsesAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	var envelope responsesEnvelope
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/responses",
		map[string]string{"Authorization": bearer(call.APIKey)},
		a.requestFor(call),
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	return a.responseFrom(call, &envelope), nil
}

func (a openAIResponsesAdapter) responseFrom(call *Call, envelope *responsesEnvelope) *Response {
	text, toolCalls, refused := splitResponsesOutput(envelope)

	return &Response{
		Text:            text,
		ToolCalls:       toolCalls,
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.InputTokens,
		OutputTokens:    envelope.Usage.OutputTokens,
		Refused:         refused,
		Truncated:       responsesTruncated(envelope),
		Reasoning:       responsesReasoningOf(envelope),
		ReasoningTokens: envelope.Usage.OutputTokensDetails.ReasoningTokens,
	}
}

// Submit starts a run the provider holds for us. Extraction of a long document
// runs for minutes, so the worker hands it over and comes back rather than
// holding a socket open across the whole thing.
func (a openAIResponsesAdapter) Submit(ctx context.Context, call *Call) (*BackgroundHandle, error) {
	body := a.requestFor(call)
	body.Background = true
	body.Store = true

	var envelope responsesEnvelope
	if err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/responses",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(envelope.ID) == "" {
		return nil, errors.New("provider accepted the run but returned no id to poll")
	}

	return &BackgroundHandle{
		ID:              envelope.ID,
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		Status:          strings.TrimSpace(envelope.Status),
	}, nil
}

func (a openAIResponsesAdapter) Poll(
	ctx context.Context,
	call *Call,
	id string,
) (*BackgroundOutcome, error) {
	var envelope responsesEnvelope
	if err := getJSON(
		ctx,
		call.Client,
		fmt.Sprintf("%s/v1/responses/%s", call.Provider.ResolvedBaseURL(), strings.TrimSpace(id)),
		map[string]string{"Authorization": bearer(call.APIKey)},
		&envelope,
	); err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(envelope.Status)
	model := stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model)

	switch raw {
	case "", "queued", "in_progress":
		return &BackgroundOutcome{
			State:           BackgroundPending,
			RawStatus:       raw,
			ModelIdentifier: model,
		}, nil
	case "completed":
		response := a.responseFrom(call, &envelope)
		if strings.TrimSpace(response.Text) == "" {
			return &BackgroundOutcome{
				State:           BackgroundFailed,
				RawStatus:       raw,
				ModelIdentifier: model,
				FailureCode:     "empty_output",
				FailureMessage:  "the run finished without producing any output",
			}, nil
		}

		return &BackgroundOutcome{
			State:           BackgroundCompleted,
			RawStatus:       raw,
			ModelIdentifier: model,
			Response:        response,
		}, nil
	default:
		code := raw
		message := fmt.Sprintf("the run ended with status %s", raw)
		if envelope.IncompleteDetails != nil &&
			strings.TrimSpace(envelope.IncompleteDetails.Reason) != "" {
			code = envelope.IncompleteDetails.Reason
			message = fmt.Sprintf("the run stopped early: %s", code)
		}
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			code = stringutils.FirstNonEmpty(strings.TrimSpace(envelope.Error.Code), code)
			message = envelope.Error.Message
		}

		return &BackgroundOutcome{
			State:           BackgroundFailed,
			RawStatus:       raw,
			ModelIdentifier: model,
			FailureCode:     code,
			FailureMessage:  message,
		}, nil
	}
}

// responsesStreamEvent is the union of the Responses API stream events this
// adapter reads. The text deltas feed the sink; the completed event carries the
// whole response, which is what the final Response is built from, so a stream
// that reaches completion is byte-for-byte what the blocking call returns.
type responsesStreamEvent struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Item     *responsesItem
	Response *struct {
		responsesEnvelope
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a openAIResponsesAdapter) Stream(
	ctx context.Context,
	call *Call,
	sink StreamSink,
) (*Response, error) {
	body := responsesRequest{
		Model:           call.Provider.Model,
		MaxOutputTokens: call.Request.MaxTokens,
		Input:           toResponsesInput(call.Request.System, call.Request.Messages),
		Tools:           toResponsesTools(call.Request.Tools),
		Stream:          true,
	}
	body.applyReasoning(call)

	stream, err := postStream(
		ctx,
		call,
		call.Provider.ResolvedBaseURL()+"/v1/responses",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var (
		thinking  strings.Builder
		text      strings.Builder
		completed *responsesEnvelope
		model     string
		refused   bool
	)

	err = readSSE(stream, func(_, data string) error {
		var event responsesStreamEvent
		if err := sonic.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode stream event: %w", err)
		}

		switch event.Type {
		case "response.created":
			if event.Response != nil {
				model = event.Response.Model
			}
		case "response.output_text.delta":
			if event.Delta != "" {
				text.WriteString(event.Delta)
				sink(event.Delta)
			}
		case "response.reasoning_summary_text.delta":
			if event.Delta != "" {
				thinking.WriteString(event.Delta)
				call.think(event.Delta)
			}
		case "response.refusal.delta":
			refused = true
		case "response.completed", "response.incomplete":
			if event.Response != nil {
				envelope := event.Response.responsesEnvelope
				completed = &envelope
			}
		case "response.failed":
			if event.Response != nil && event.Response.Error != nil {
				return streamError(event.Response.Error.Code, event.Response.Error.Message)
			}
			return streamError("", "response failed")
		case "error":
			if event.Error != nil {
				return streamError(event.Error.Code, event.Error.Message)
			}
			return streamError("", "")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if completed == nil {
		// The stream closed without a terminal event. The text that arrived is
		// still the model's answer, so it is returned rather than discarded; tool
		// calls cannot be trusted without the completed output, so none are.
		return &Response{
			Text:            text.String(),
			ModelIdentifier: stringutils.FirstNonEmpty(model, call.Provider.Model),
			Refused:         refused,
			Reasoning:       textReasoning(thinking.String()),
		}, nil
	}

	finalText, toolCalls, finalRefused := splitResponsesOutput(completed)
	reasoning := responsesReasoningOf(completed)
	if reasoning == nil {
		reasoning = textReasoning(thinking.String())
	}

	return &Response{
		Text:            stringutils.FirstNonEmpty(finalText, text.String()),
		ToolCalls:       toolCalls,
		ModelIdentifier: stringutils.FirstNonEmpty(completed.Model, model, call.Provider.Model),
		InputTokens:     completed.Usage.InputTokens,
		OutputTokens:    completed.Usage.OutputTokens,
		Refused:         refused || finalRefused,
		Truncated:       responsesTruncated(completed),
		Reasoning:       reasoning,
		ReasoningTokens: completed.Usage.OutputTokensDetails.ReasoningTokens,
	}, nil
}

// responsesReasoningOf gathers a response's reasoning items into one trace:
// the summaries to read, the first item's id and encrypted content to replay.
func responsesReasoningOf(envelope *responsesEnvelope) *ReasoningTrace {
	if envelope == nil {
		return nil
	}
	var trace ReasoningTrace
	var text strings.Builder
	found := false
	for idx := range envelope.Output {
		item := &envelope.Output[idx]
		if item.Type != "reasoning" {
			continue
		}
		found = true
		for _, part := range item.Summary {
			if text.Len() > 0 && part.Text != "" {
				text.WriteString("\n\n")
			}
			text.WriteString(part.Text)
		}
		if trace.Signature == "" {
			trace.Signature = item.ID
			trace.Encrypted = item.EncryptedContent
		}
	}
	if !found {
		return nil
	}
	trace.Text = text.String()

	return &trace
}

// replayReasoning is the reasoning item a previous assistant turn's function
// calls must follow. Without it the calls are refused as orphans.
func replayReasoning(trace *ReasoningTrace) []responsesItem {
	if trace == nil || trace.Signature == "" {
		return nil
	}

	return []responsesItem{{
		Type:             "reasoning",
		ID:               trace.Signature,
		EncryptedContent: trace.Encrypted,
		Summary:          []responsesSummaryPart{},
	}}
}

// responsesTruncated reads the Responses API's two ways of saying the output
// limit was hit: the response status, and the reason on an incomplete one.
func responsesTruncated(envelope *responsesEnvelope) bool {
	if envelope == nil {
		return false
	}
	if envelope.IncompleteDetails != nil && envelope.IncompleteDetails.Reason == "max_output_tokens" {
		return true
	}

	return envelope.Status == "incomplete"
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
			items = append(items, replayReasoning(msg.Reasoning)...)
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
			args, argsErr := decodeArguments(item.Arguments)
			toolCalls = append(toolCalls, ToolCall{
				ID:             item.CallID,
				Name:           item.Name,
				Arguments:      args,
				ArgumentsError: argsErr,
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
