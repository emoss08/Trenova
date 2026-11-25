package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A turn that asked nothing anyone saw has not said what it needed to, so
// an empty reply after it is asked for again like any other.
func TestRun_AQuestionNobodySawDoesNotEndTheTurnInSilence(t *testing.T) {
	t.Parallel()

	result, completion, _ := runReplies(
		t,
		runtimetest.ToolTurn(
			"ask_user",
			map[string]any{"question": "Which customer, pickup and delivery?"},
		),
		runtimetest.TextTurn(""),
		runtimetest.TextTurn("I need the customer the load is for before I can book it."),
	)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "several values at once")
	assert.Equal(t, 3, completion.CallCount)
	assert.Equal(t, "I need the customer the load is for before I can book it.", result.Reply)
}

func TestRun_AQuestionRefusedAsAskedCanBeAskedAgainProperly(t *testing.T) {
	t.Parallel()

	question := "What does the shipment still need?"
	result, _, _ := runReplies(
		t,
		runtimetest.ToolTurn("ask_user", map[string]any{
			"question": question,
			"options":  []any{map[string]any{"value": "a", "label": "A"}},
			"fields":   runtimetest.AskFieldArgs(),
		}),
		runtimetest.ToolTurn(
			"ask_user",
			map[string]any{"question": question, "fields": runtimetest.AskFieldArgs()},
		),
		runtimetest.TextTurn(""),
	)

	require.Len(t, result.Messages, 6, "the question, two asks with their answers, and the reply")
	assert.True(t, result.Messages[2].ToolFailed)
	assert.False(t, result.Messages[4].ToolFailed, "the form is shown, not refused as repeated")
	assert.Contains(t, result.Messages[4].Content, toolresult.UntrustedOpenTag)
	assert.Empty(t, result.Reply)
}

// A reply that closes on a question to the person is sent back once, to be
// asked with ask_user: the person answers it with a click, pinned above the
// composer, instead of reading the reply for what they were asked.
func TestRun_AQuestionWrittenInTheReplyIsAskedWithAskUser(t *testing.T) {
	t.Parallel()

	result, completion, _ := runReplies(
		t,
		runtimetest.TextTurn("Batch 1 is planned: 8 picks.\n\nShould I move on to batch 2?"),
		runtimetest.ToolTurn("ask_user", map[string]any{
			"question": "Should I move on to batch 2?",
			"options": []any{
				map[string]any{"label": "Yes, plan batch 2"},
				map[string]any{"label": "No, stop here"},
			},
		}),
	)

	assert.Equal(t, 2, completion.CallCount)
	shown := false
	for _, message := range result.Messages {
		if message.ToolName == "ask_user" && !message.ToolFailed {
			shown = true
		}
	}
	assert.True(t, shown, "the question reaches the person with ask_user")
	assert.NotContains(t, result.Reply, "Should I move on to batch 2?")
}
