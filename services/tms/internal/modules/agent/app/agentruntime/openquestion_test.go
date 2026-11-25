package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const batchQuestion = "Should I start planning drivers for the first batch of 50 Delayed shipments?"

func openQuestionRequest(trenovaWritten bool) *agentruntimeports.RunRequest {
	definition := runtimetest.Definition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute

	return &agentruntimeports.RunRequest{
		Definition:     definition,
		Actor:          runtimetest.Actor(),
		Input:          "Approved agent_create_memory, and it ran.",
		TrenovaWritten: trenovaWritten,
		OpenQuestion:   batchQuestion,
	}
}

/*
The agent saved a memory as a proposal and asked whether to start planning
drivers for the first batch of Delayed shipments. The person approved the
memory; the follow-up turn said "You chose 'Yes, start with batch 1 of 50'"
and started dispatch calls before they had answered. A follow-up that arrives
under an open question now runs no tool: each call is refused, naming the
question as still waiting, and the turn ends on a report.
*/
func TestRun_DecisionFollowUpUnderAnOpenQuestionRunsNoTool(t *testing.T) {
	t.Parallel()

	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		runtimetest.TextTurn("The memory was saved. Your question about batch 1 is still waiting."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)

	result, err := rt.Run(t.Context(), openQuestionRequest(true))
	require.NoError(t, err)

	assert.Zero(t, action.Calls, "the work the question asks about must not start")
	assert.Empty(t, result.Actions, "nothing is proposed either")
	refusal := toolResultFor(result.Messages, "dispatch_assign_move")
	require.NotNil(t, refusal)
	assert.True(t, refusal.ToolFailed)
	assert.Contains(t, refusal.Content, batchQuestion)
	assert.Contains(t, refusal.Content, "their answer has not arrived")
	assert.Contains(t, result.Reply, "still waiting")
}

// The state a durable turn is carried in keeps the hold, so a turn restored
// on a worker refuses as the one opened in process would.
func TestTurnState_CarriesTheOpenQuestion(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, nil, nil, nil)
	state := rt.OpenTurn(t.Context(), openQuestionRequest(true)).State()
	assert.Equal(t, batchQuestion, state.OpenQuestion)

	restored := rt.RestoreTurn(openQuestionRequest(true), &state)
	assert.Equal(t, batchQuestion, restored.openQuestion)

	assert.Empty(t, rt.OpenTurn(t.Context(), openQuestionRequest(false)).State().OpenQuestion,
		"a turn the person opened answers what they said, and is not held")
}

func toolResultFor(messages []conversation.Message, tool string) *conversation.Message {
	for idx := range messages {
		if messages[idx].Role == conversation.RoleTool && messages[idx].ToolName == tool {
			return &messages[idx]
		}
	}

	return nil
}
