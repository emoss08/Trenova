package palettejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/contrib/workflowstreams"
)

// streamEvents puts an answer on its stream in the events the palette's
// reader already follows: the same text, steps and records a Desk turn sends.
// Each read that finishes is also handed to record, which keeps it, so why an
// answer came out as it did can be read afterwards.
type streamEvents struct {
	topic  *workflowstreams.TopicHandle
	record func(*palettequestion.ToolCall)

	// model is the model that asked for the reads now running; started is
	// when each call began, by call id; reads counts the question's reads.
	model   string
	started map[string]int64
	reads   int
}

func (e *streamEvents) publish(event string, data any, force bool) {
	e.topic.Publish(
		temporaltype.StreamItem{Event: event, Data: data, At: timeutils.NowUnix()},
		force,
	)
}

func (e *streamEvents) Delta(text string) {
	e.publish(assistantstreamports.AssistantEventDelta, assistantstreamports.AssistantDeltaEvent{
		Text: text,
	}, false)
}

func (e *streamEvents) Message(content string, calls []llmports.ToolCall, model string) {
	e.model = model
	records := make([]conversation.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		records = append(records, conversation.ToolCallRecord{
			ID: call.ID, Name: call.Name, Arguments: call.Arguments, Effect: agenttooltrust.ToolEffectLookup,
		})
	}
	e.publish(
		assistantstreamports.AssistantEventMessage,
		assistantstreamports.AssistantMessageEvent{
			Content: content, ToolCalls: records, Model: model,
		},
		true,
	)
}

func (e *streamEvents) ToolStarted(call llmports.ToolCall) {
	if e.started == nil {
		e.started = make(map[string]int64, 4)
	}
	e.started[call.ID] = time.Now().UnixMilli()
	e.publish(
		assistantstreamports.AssistantEventToolStarted,
		assistantstreamports.AssistantToolStartedEvent{
			CallID:    call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
			Effect:    agenttooltrust.ToolEffectLookup,
		},
		true,
	)
}

func (e *streamEvents) ToolFinished(call llmports.ToolCall, result llmports.NovaToolResult) {
	e.publish(
		assistantstreamports.AssistantEventToolFinished,
		assistantstreamports.AssistantToolFinishedEvent{
			CallID:  call.ID,
			Name:    call.Name,
			Failed:  result.Failed,
			Content: result.Content,
			Effect:  agenttooltrust.ToolEffectLookup,
			Summary: result.Summary,
			Records: result.Records,
		},
		true,
	)

	if e.record != nil {
		e.record(e.toolCall(call, result))
	}
}

// toolCall is a finished read as the question's record keeps it: what the
// model called and the read it went to, how it ended and why, never what it
// returned.
func (e *streamEvents) toolCall(
	call llmports.ToolCall,
	result llmports.NovaToolResult,
) *palettequestion.ToolCall {
	now := time.Now().UnixMilli()
	started, ok := e.started[call.ID]
	if !ok {
		started = now
	}
	e.reads++

	kept := &palettequestion.ToolCall{
		CallID:      call.ID,
		Sequence:    e.reads,
		Model:       e.model,
		Tool:        call.Name,
		Arguments:   call.Arguments,
		Read:        result.Read,
		Params:      result.Params,
		Outcome:     palettequestion.ToolCallRan,
		Summary:     result.Summary,
		Records:     len(result.Records),
		StartedAtMs: started,
		DurationMs:  max(now-started, 0),
	}
	switch {
	case result.Refused:
		kept.Outcome = palettequestion.ToolCallRefused
		kept.Reason = result.Content
	case result.Failed:
		kept.Outcome = palettequestion.ToolCallFailed
		kept.Reason = result.Content
	}
	if kept.Arguments == nil {
		kept.Arguments = map[string]any{}
	}
	if kept.Params == nil {
		kept.Params = map[string]any{}
	}

	return kept
}

var _ llmports.NovaEvents = (*streamEvents)(nil)
