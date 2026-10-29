package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
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
			askuser.Name,
			map[string]any{"question": "Which customer, pickup and delivery?"},
		),
		runtimetest.TextTurn(""),
		runtimetest.TextTurn("Which customer is the load for?"),
	)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "several values at once")
	assert.Equal(t, 3, completion.CallCount)
	assert.Equal(t, "Which customer is the load for?", result.Reply)
}

func TestRun_AQuestionRefusedAsAskedCanBeAskedAgainProperly(t *testing.T) {
	t.Parallel()

	question := "What does the shipment still need?"
	result, _, _ := runReplies(
		t,
		runtimetest.ToolTurn(askuser.Name, map[string]any{
			"question": question,
			"options":  []any{map[string]any{"value": "a", "label": "A"}},
			"fields":   runtimetest.AskFieldArgs(),
		}),
		runtimetest.ToolTurn(
			askuser.Name,
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
