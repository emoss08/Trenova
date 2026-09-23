package services

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Another agent's events reach the reader of the turn that delegated tagged
// with the task. Its streamed words and restarts travel under names of their
// own, because a reader applies a plain delta to the reply it is showing and
// a plain restart discards it; a refusal of its answer is reported when the
// task finishes instead.
func TestDelegateScope_TagsWhatAReaderMustNotApplyToTheReply(t *testing.T) {
	t.Parallel()

	scope := DelegateScope{AgentID: pulid.MustNew("agdef_"), DelegateCallID: "call_1"}

	delta, shown := scope.Tag(StreamEvent{Event: AssistantEventDelta,
		Data: AssistantDeltaEvent{Text: "Saved."}})
	require.True(t, shown)
	assert.Equal(t, AssistantEventDelegateDelta, delta.Event)
	encoded, err := sonic.MarshalString(delta.Data)
	require.NoError(t, err)
	assert.JSONEq(t, `{"agentId":"`+scope.AgentID.String()+
		`","delegateCallId":"call_1","text":"Saved."}`, encoded)

	thinking, _ := scope.Tag(StreamEvent{Event: AssistantEventReasoning,
		Data: AssistantReasoningEvent{Text: "Hmm."}})
	assert.Equal(t, AssistantEventDelegateReasoning, thinking.Event)

	restart, _ := scope.Tag(StreamEvent{Event: AssistantEventRetrying,
		Data: AssistantRetryingEvent{Attempt: 1}})
	assert.Equal(t, AssistantEventDelegateRetrying, restart.Event)
	assert.Equal(t, "call_1", restart.Data.(AssistantRetryingEvent).DelegateCallID)

	started, _ := scope.Tag(StreamEvent{Event: AssistantEventToolStarted,
		Data: AssistantToolStartedEvent{CallID: "call_d1", Name: "create_report"}})
	assert.Equal(t, AssistantEventToolStarted, started.Event, "tool events keep their names")
	assert.Equal(t, scope.AgentID, started.Data.(AssistantToolStartedEvent).AgentID)

	finished, _ := scope.Tag(StreamEvent{Event: AssistantEventToolFinished,
		Data: AssistantToolFinishedEvent{CallID: "call_d1"}})
	assert.Equal(t, "call_1", finished.Data.(AssistantToolFinishedEvent).DelegateCallID)

	message, _ := scope.Tag(StreamEvent{Event: AssistantEventMessage,
		Data: AssistantMessageEvent{Content: "Working."}})
	assert.Equal(t, "call_1", message.Data.(AssistantMessageEvent).DelegateCallID)

	_, shown = scope.Tag(StreamEvent{Event: AssistantEventRefused,
		Data: AssistantRefusedEvent{Message: "Withheld."}})
	assert.False(t, shown)

	own, shown := DelegateScope{}.Tag(StreamEvent{Event: AssistantEventDelta,
		Data: AssistantDeltaEvent{Text: "Mine."}})
	assert.True(t, shown)
	assert.Equal(t, AssistantEventDelta, own.Event, "the turn's own events are untouched")
}
