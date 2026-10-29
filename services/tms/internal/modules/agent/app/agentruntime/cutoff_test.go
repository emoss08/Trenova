package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cutOffEffects struct {
	*localEffects

	events []kernelports.StreamEvent
}

func (fx *cutOffEffects) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func cutMidCall(limit int) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		Text:            "I'll transfer all 21 now.",
		Truncated:       true,
		OutputLimit:     limit,
		OutputTokens:    limit,
		CutOffCall:      &llmports.CutOffToolCall{Name: "billing_transfer"},
		ModelIdentifier: "z-ai/glm-5.3",
	}
}

func cutBeforeAnswering(limit int) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		Truncated:       true,
		OutputLimit:     limit,
		OutputTokens:    limit,
		ModelIdentifier: "z-ai/glm-5.3",
	}
}

type cutOffRun struct {
	result     *agentruntimeports.RunResult
	completion *scriptedCompletion
	fx         *cutOffEffects
	tool       *agenttest.StubActionTool
}

func runCutOff(
	t *testing.T,
	turns ...*llmports.ChatCompletionResult,
) cutOffRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	tool := &agenttest.StubActionTool{
		ToolName: "billing_transfer",
		Tier:     agent.TierPropose,
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	fx := &cutOffEffects{}
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("billing_transfer"),
		Actor:      runtimetest.Actor(),
		Input:      "Transfer all of them to billing.",
	}), fx)
	require.NoError(t, err)

	return cutOffRun{result: result, completion: completion, fx: fx, tool: tool}
}

func retryingReasons(events []kernelports.StreamEvent) []string {
	reasons := make([]string, 0, 1)
	for _, event := range events {
		if data, ok := event.Data.(assistantstreamports.AssistantRetryingEvent); ok {
			reasons = append(reasons, data.Reason)
		}
	}

	return reasons
}

func TestCutOffCall_IsAskedAgainWithTwiceTheRoom(t *testing.T) {
	t.Parallel()

	run := runCutOff(t,
		cutMidCall(1024),
		runtimetest.ToolTurn("billing_transfer", map[string]any{"shipmentIds": []any{"shp_1"}}),
		runtimetest.TextTurn("Proposed the transfer of shp_1."),
	)

	require.Equal(t, 3, run.completion.CallCount)
	assert.Zero(t, run.completion.Requests[0].MaxTokens, "the first call uses the provider's own")
	assert.Equal(t, 2048, run.completion.Requests[1].MaxTokens)
	assert.Equal(t, 2048, run.completion.Requests[2].MaxTokens,
		"the rest of the turn keeps the room it needed")
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, "billing_transfer", run.result.Actions[0].ToolName)
	assert.Equal(t, "Proposed the transfer of shp_1.", run.result.Reply)
	assert.False(t, run.result.Truncated)
	assert.Equal(t, []string{loopguard.CutOffRetryReason}, retryingReasons(run.fx.events),
		"the reader drops the markup it watched stream")
	for _, message := range run.result.Messages {
		assert.NotContains(t, message.Content, "I'll transfer all 21 now.",
			"the cut attempt is not kept in the conversation")
	}
}

func TestCutOffCall_StillCutSaysItRanOutOfRoomAndNamesTheLimit(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, cutMidCall(1024), cutMidCall(2048))

	require.Equal(t, 2, run.completion.CallCount, "asked again once, not again and again")
	assert.Empty(t, run.result.Actions)
	assert.Zero(t, run.tool.Calls)
	assert.True(t, run.result.Truncated)
	reply := run.result.Reply
	assert.True(t, strings.HasPrefix(reply, "I'll transfer all 21 now."))
	assert.Contains(t, reply, "ran out of room")
	assert.Contains(t, reply, "billing_transfer")
	assert.Contains(t, reply, "was not filed")
	assert.Contains(t, reply, "1024")
	assert.Contains(t, reply, "2048")
	assert.Contains(t, reply, "Max tokens")
	assert.NotContains(t, reply, truncationNotice)
	assert.NotContains(t, reply, "Ask again")
}

func TestCutOffCall_AReplyThatSpentItsBudgetThinkingIsAskedAgain(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, cutBeforeAnswering(1024), runtimetest.TextTurn("21 shipments can go."))

	require.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, 2048, run.completion.Requests[1].MaxTokens)
	assert.Equal(t, "21 shipments can go.", run.result.Reply)
}

func TestCutOffCall_ABrokenNativeCallIsAskedAgain(t *testing.T) {
	t.Parallel()

	broken := &llmports.ChatCompletionResult{
		Truncated:   true,
		OutputLimit: 4096,
		ToolCalls: []llmports.ToolCall{{
			ID:             "call_1",
			Name:           "billing_transfer",
			Arguments:      map[string]any{},
			ArgumentsError: "unexpected end of JSON input",
		}},
	}
	run := runCutOff(t, broken, broken)

	require.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, 8192, run.completion.Requests[1].MaxTokens)
	assert.Contains(t, run.result.Reply, "billing_transfer")
	assert.Contains(t, run.result.Reply, "ran out of room")
	assert.Zero(t, run.result.ToolCallsUsed, "a call that was never whole spends no budget")
}

func TestCutOffCall_TheRetryIsCappedByTheRuntimeCeiling(t *testing.T) {
	t.Parallel()

	run := runCutOff(
		t,
		cutMidCall(loopguard.CutOffRetryCeiling),
		runtimetest.TextTurn("never asked"),
	)

	require.Equal(t, 1, run.completion.CallCount, "no more room to give, so it is not asked again")
	assert.Contains(t, run.result.Reply, "ran out of room")

	capped := runCutOff(t, cutMidCall(20000), runtimetest.TextTurn("Done."))
	require.Equal(t, 2, capped.completion.CallCount)
	assert.Equal(t, loopguard.CutOffRetryCeiling, capped.completion.Requests[1].MaxTokens)
}

func TestCutOffCall_ALongTextAnswerKeepsTheOldNotice(t *testing.T) {
	t.Parallel()

	long := &llmports.ChatCompletionResult{
		Text:        "| Shipment | Amount |\n| shp_1 | 100 |",
		Truncated:   true,
		OutputLimit: 1024,
	}
	run := runCutOff(t, long)

	require.Equal(t, 1, run.completion.CallCount)
	assert.True(t, strings.HasSuffix(run.result.Reply, truncationNotice))
}
