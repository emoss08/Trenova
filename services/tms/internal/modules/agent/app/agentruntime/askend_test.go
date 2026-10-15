package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type askEndEffects struct {
	*localEffects

	supports bool
	asked    []string
}

func (fx *askEndEffects) Supports(change string) bool {
	fx.asked = append(fx.asked, change)
	if change == changeEndOnAsk {
		return fx.supports
	}

	return true
}

type askEndRun struct {
	result     *agentports.RunResult
	completion *scriptedCompletion
	fx         *askEndEffects
}

func runAskEnd(
	t *testing.T,
	supports bool,
	turns ...*agentports.ChatCompletionResult,
) askEndRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &askEndEffects{supports: supports}
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "what is on my dashboard",
	}), fx)
	require.NoError(t, err)

	return askEndRun{result: result, completion: completion, fx: fx}
}

func askTurn(question string) *agentports.ChatCompletionResult {
	return toolTurn(askUserName, map[string]any{"question": question, "allowOther": true})
}

func TestRun_EndsTheTurnOnceAQuestionIsShown(t *testing.T) {
	t.Parallel()

	run := runAskEnd(t, true, askTurn("Which dashboard?"))

	assert.Equal(t, 1, run.completion.CallCount,
		"the model told to end its turn is not asked to write again")
	assert.Empty(t, run.result.Reply)
	require.NotEmpty(t, run.result.Messages)
	last := run.result.Messages[len(run.result.Messages)-1]
	assert.Empty(t, last.Content)
	assert.Contains(t, run.fx.asked, changeEndOnAsk)
}

func TestRun_EndsTheTurnAfterEveryCallOfTheReplyThatAsked(t *testing.T) {
	t.Parallel()

	turn := askTurn("Which dashboard?")
	turn.ToolCalls = append(turn.ToolCalls, agentports.ToolCall{
		ID: "call_2", Name: "list_dashboards", Arguments: map[string]any{},
	})

	run := runAskEnd(t, true, turn)

	assert.Equal(t, 1, run.completion.CallCount)
	results := 0
	for _, message := range run.result.Messages {
		if message.ToolCallID != "" {
			results++
		}
	}
	assert.Equal(t, 2, results, "each call of the reply is answered before the turn ends")
}

func TestRun_AnOlderTurnAsksTheModelAgainAfterAQuestion(t *testing.T) {
	t.Parallel()

	run := runAskEnd(t, false, askTurn("Which dashboard?"), textTurn(""))

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.result.Reply)
}

func TestRun_KeepsWorkingAfterAQuestionThatWasNotShown(t *testing.T) {
	t.Parallel()

	run := runAskEnd(t, true,
		toolTurn(askUserName, map[string]any{"question": "  "}),
		textTurn("Your dashboard has three tiles."),
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, "Your dashboard has three tiles.", run.result.Reply)
	assert.NotContains(t, run.fx.asked, changeEndOnAsk,
		"a turn that showed no question records no marker")
}
