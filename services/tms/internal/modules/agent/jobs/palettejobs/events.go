package palettejobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/contrib/workflowstreams"
)

// streamEvents puts an answer on its stream in the events the palette's
// reader already follows: the same text, steps and records a Desk turn sends.
type streamEvents struct {
	topic *workflowstreams.TopicHandle
}

func (e *streamEvents) publish(event string, data any, force bool) {
	e.topic.Publish(temporaltype.StreamItem{Event: event, Data: data, At: timeutils.NowUnix()}, force)
}

func (e *streamEvents) Delta(text string) {
	e.publish(assistantstreamports.AssistantEventDelta, assistantstreamports.AssistantDeltaEvent{
		Text: text,
	}, false)
}

func (e *streamEvents) Message(content string, calls []llmports.ToolCall, model string) {
	records := make([]conversation.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		records = append(records, conversation.ToolCallRecord{
			ID: call.ID, Name: call.Name, Arguments: call.Arguments, Effect: agenttooltrust.ToolEffectLookup,
		})
	}
	e.publish(assistantstreamports.AssistantEventMessage, assistantstreamports.AssistantMessageEvent{
		Content: content, ToolCalls: records, Model: model,
	}, true)
}

func (e *streamEvents) ToolStarted(call llmports.ToolCall) {
	e.publish(assistantstreamports.AssistantEventToolStarted, assistantstreamports.AssistantToolStartedEvent{
		CallID:    call.ID,
		Name:      call.Name,
		Arguments: call.Arguments,
		Effect:    agenttooltrust.ToolEffectLookup,
	}, true)
}

func (e *streamEvents) ToolFinished(call llmports.ToolCall, result llmports.NovaToolResult) {
	e.publish(assistantstreamports.AssistantEventToolFinished, assistantstreamports.AssistantToolFinishedEvent{
		CallID:  call.ID,
		Name:    call.Name,
		Failed:  result.Failed,
		Content: result.Content,
		Effect:  agenttooltrust.ToolEffectLookup,
		Summary: result.Summary,
		Records: result.Records,
	}, true)
}

var _ llmports.NovaEvents = (*streamEvents)(nil)
