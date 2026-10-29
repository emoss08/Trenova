package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type askEndRun struct {
	result     *agentruntimeports.RunResult
	completion *scriptedCompletion
}

func runAskEnd(
	t *testing.T,
	turns ...*llmports.ChatCompletionResult,
) askEndRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "what is on my dashboard",
	}), fx)
	require.NoError(t, err)

	return askEndRun{result: result, completion: completion}
}

func askTurn(question string) *llmports.ChatCompletionResult {
	return runtimetest.ToolTurn(
		askuser.Name,
		map[string]any{"question": question, "allowOther": true},
	)
}

func TestRun_EndsTheTurnOnceAQuestionIsShown(t *testing.T) {
	t.Parallel()

	run := runAskEnd(t, askTurn("Which dashboard?"))

	assert.Equal(t, 1, run.completion.CallCount,
		"the model told to end its turn is not asked to write again")
	assert.Empty(t, run.result.Reply)
	require.NotEmpty(t, run.result.Messages)
	last := run.result.Messages[len(run.result.Messages)-1]
	assert.Empty(t, last.Content)
}

func TestRun_EndsTheTurnAfterEveryCallOfTheReplyThatAsked(t *testing.T) {
	t.Parallel()

	turn := askTurn("Which dashboard?")
	turn.ToolCalls = append(turn.ToolCalls, llmports.ToolCall{
		ID: "call_2", Name: "report_list_dashboards", Arguments: map[string]any{},
	})

	run := runAskEnd(t, turn)

	assert.Equal(t, 1, run.completion.CallCount)
	results := 0
	for _, message := range run.result.Messages {
		if message.ToolCallID != "" {
			results++
		}
	}
	assert.Equal(t, 2, results, "each call of the reply is answered before the turn ends")
}

func TestRun_KeepsWorkingAfterAQuestionThatWasNotShown(t *testing.T) {
	t.Parallel()

	run := runAskEnd(t,
		runtimetest.ToolTurn(askuser.Name, map[string]any{"question": "  "}),
		runtimetest.TextTurn("Your dashboard has three tiles."),
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, "Your dashboard has three tiles.", run.result.Reply)
}
