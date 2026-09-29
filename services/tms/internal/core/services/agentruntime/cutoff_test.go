package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cutOffEffects struct {
	*localEffects

	supports bool
	asked    []string
	events   []serviceports.StreamEvent
}

func (fx *cutOffEffects) Emit(event serviceports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *cutOffEffects) Supports(change string) bool {
	fx.asked = append(fx.asked, change)
	if change == changeCutOffCallRetry {
		return fx.supports
	}

	return true
}

func cutMidCall(limit int) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		Text:            "I'll transfer all 21 now.",
		Truncated:       true,
		OutputLimit:     limit,
		OutputTokens:    limit,
		CutOffCall:      &serviceports.CutOffToolCall{Name: "transfer_to_billing"},
		ModelIdentifier: "z-ai/glm-5.3",
	}
}

func cutBeforeAnswering(limit int) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		Truncated:       true,
		OutputLimit:     limit,
		OutputTokens:    limit,
		ModelIdentifier: "z-ai/glm-5.3",
	}
}

type cutOffRun struct {
	result     *serviceports.RunResult
	completion *scriptedCompletion
	fx         *cutOffEffects
	tool       *agentruntimetest.StubActionTool
}

func runCutOff(
	t *testing.T,
	supports bool,
	turns ...*serviceports.ChatCompletionResult,
) cutOffRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	tool := &agentruntimetest.StubActionTool{
		ToolName: "transfer_to_billing",
		Tier:     agent.TierPropose,
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	fx := &cutOffEffects{supports: supports}
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(serviceports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("transfer_to_billing"),
		Actor:      testActor(),
		Input:      "Transfer all of them to billing.",
	}), fx)
	require.NoError(t, err)

	return cutOffRun{result: result, completion: completion, fx: fx, tool: tool}
}

func retryingReasons(events []serviceports.StreamEvent) []string {
	reasons := make([]string, 0, 1)
	for _, event := range events {
		if data, ok := event.Data.(serviceports.AssistantRetryingEvent); ok {
			reasons = append(reasons, data.Reason)
		}
	}

	return reasons
}

func TestCutOffCall_IsAskedAgainWithTwiceTheRoom(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, true,
		cutMidCall(1024),
		toolTurn("transfer_to_billing", map[string]any{"shipmentIds": []any{"shp_1"}}),
		textTurn("Proposed the transfer of shp_1."),
	)

	require.Equal(t, 3, run.completion.CallCount)
	assert.Zero(t, run.completion.Requests[0].MaxTokens, "the first call uses the provider's own")
	assert.Equal(t, 2048, run.completion.Requests[1].MaxTokens)
	assert.Equal(t, 2048, run.completion.Requests[2].MaxTokens,
		"the rest of the turn keeps the room it needed")
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, "transfer_to_billing", run.result.Actions[0].ToolName)
	assert.Equal(t, "Proposed the transfer of shp_1.", run.result.Reply)
	assert.False(t, run.result.Truncated)
	assert.Equal(t, []string{cutOffRetryReason}, retryingReasons(run.fx.events),
		"the reader drops the markup it watched stream")
	for _, message := range run.result.Messages {
		assert.NotContains(t, message.Content, "I'll transfer all 21 now.",
			"the cut attempt is not kept in the conversation")
	}
}

func TestCutOffCall_StillCutSaysItRanOutOfRoomAndNamesTheLimit(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, true, cutMidCall(1024), cutMidCall(2048))

	require.Equal(t, 2, run.completion.CallCount, "asked again once, not again and again")
	assert.Empty(t, run.result.Actions)
	assert.Zero(t, run.tool.Calls)
	assert.True(t, run.result.Truncated)
	reply := run.result.Reply
	assert.True(t, strings.HasPrefix(reply, "I'll transfer all 21 now."))
	assert.Contains(t, reply, "ran out of room")
	assert.Contains(t, reply, "transfer_to_billing")
	assert.Contains(t, reply, "was not filed")
	assert.Contains(t, reply, "1024")
	assert.Contains(t, reply, "2048")
	assert.Contains(t, reply, "Max tokens")
	assert.NotContains(t, reply, truncationNotice)
	assert.NotContains(t, reply, "Ask again")
}

func TestCutOffCall_AReplyThatSpentItsBudgetThinkingIsAskedAgain(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, true, cutBeforeAnswering(1024), textTurn("21 shipments can go."))

	require.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, 2048, run.completion.Requests[1].MaxTokens)
	assert.Equal(t, "21 shipments can go.", run.result.Reply)
}

func TestCutOffCall_ABrokenNativeCallIsAskedAgain(t *testing.T) {
	t.Parallel()

	broken := &serviceports.ChatCompletionResult{
		Truncated:   true,
		OutputLimit: 4096,
		ToolCalls: []serviceports.ToolCall{{
			ID:             "call_1",
			Name:           "transfer_to_billing",
			Arguments:      map[string]any{},
			ArgumentsError: "unexpected end of JSON input",
		}},
	}
	run := runCutOff(t, true, broken, broken)

	require.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, 8192, run.completion.Requests[1].MaxTokens)
	assert.Contains(t, run.result.Reply, "transfer_to_billing")
	assert.Contains(t, run.result.Reply, "ran out of room")
	assert.Zero(t, run.result.ToolCallsUsed, "a call that was never whole spends no budget")
}

func TestCutOffCall_TheRetryIsCappedByTheRuntimeCeiling(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, true, cutMidCall(cutOffRetryCeiling), textTurn("never asked"))

	require.Equal(t, 1, run.completion.CallCount, "no more room to give, so it is not asked again")
	assert.Contains(t, run.result.Reply, "ran out of room")

	capped := runCutOff(t, true, cutMidCall(20000), textTurn("Done."))
	require.Equal(t, 2, capped.completion.CallCount)
	assert.Equal(t, cutOffRetryCeiling, capped.completion.Requests[1].MaxTokens)
}

func TestCutOffCall_ALongTextAnswerKeepsTheOldNotice(t *testing.T) {
	t.Parallel()

	long := &serviceports.ChatCompletionResult{
		Text:        "| Shipment | Amount |\n| shp_1 | 100 |",
		Truncated:   true,
		OutputLimit: 1024,
	}
	run := runCutOff(t, true, long)

	require.Equal(t, 1, run.completion.CallCount)
	assert.True(t, strings.HasSuffix(run.result.Reply, truncationNotice))
	assert.NotContains(t, run.fx.asked, changeCutOffCallRetry,
		"a turn that never meets a cut call records no marker")
}

func TestCutOffCall_AnExecutionStartedBeforeTheChangeKeepsItsShape(t *testing.T) {
	t.Parallel()

	run := runCutOff(t, false, cutMidCall(1024), textTurn("never asked"))

	require.Equal(t, 1, run.completion.CallCount)
	assert.Equal(t, "I'll transfer all 21 now."+truncationNotice, run.result.Reply)
	assert.Contains(t, run.fx.asked, changeCutOffCallRetry)
	assert.Empty(t, retryingReasons(run.fx.events))
}
