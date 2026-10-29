package agentruntime

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The transcript behind this has one report parameter rejected five times, the
// identical payload each time, with a different explanation narrated before
// each attempt. The tool budget went on it and the thread filled with the same
// red row.
func TestRun_WillNotRepeatACallThatAlreadyFailedUnchanged(t *testing.T) {
	t.Parallel()

	args := map[string]any{"statuses": map[string]any{"item": []any{"Completed"}}}
	tool := queryTool("report_run", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_run", args),
		runtimetest.ToolTurn("report_run", args),
		runtimetest.TextTurn("I could not start it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_run"),
		Actor:      runtimetest.Actor(),
		Input:      "run the unbilled report",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls, "the identical call is not run a second time")

	var refusals int
	for _, message := range result.Messages {
		if strings.Contains(message.Content, "was already made in this turn and failed") {
			refusals++
			assert.Contains(t, message.Content, "expected a list, got map",
				"the refusal repeats the original error, which the model did not absorb")
		}
	}
	assert.Equal(t, 1, refusals)
}

/*
A billing turn sent the same refused transfer five more times after the first
refusal, alternating between two variants, and spent its whole tool budget
before answering. Once a turn has sent failed calls again unchanged twice, it
stops calling tools and answers from what it has, told why: the person hears
what was refused in seconds instead of a minute later.
*/
func TestRun_StopsCallingToolsAfterRepeatedFailedCalls(t *testing.T) {
	t.Parallel()

	args := map[string]any{"statuses": map[string]any{"item": []any{"Completed"}}}
	tool := queryTool("report_run", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_run", args),
		runtimetest.ToolTurn("report_run", args),
		runtimetest.ToolTurn("report_run", args),
		runtimetest.TextTurn("The report refused the status filter, so it did not run."),
		runtimetest.ToolTurn("report_run", args),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_run"),
		Actor:      runtimetest.Actor(),
		Input:      "run the unbilled report",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls)
	assert.Equal(t, 3, result.ToolCallsUsed, "the turn stopped well short of its budget")
	assert.Equal(t, "The report refused the status filter, so it did not run.", result.Reply)
	require.Len(t, completion.Requests, 4)
	final := completion.Requests[3]
	assert.True(t, final.ToolChoiceNone, "the answer is asked for with tool calls forbidden")
	assert.Equal(t, ReaskStuck, final.Reask)
	assert.Empty(t, completion.Requests[2].Reask)
	assert.Equal(t, specNames(completion.Requests[2].Tools), specNames(final.Tools),
		"the tools stay declared so the cached prefix holds")
	assert.Equal(t, loopguard.StuckNote, final.Messages[len(final.Messages)-1].Content)
}

// The retry that was wanted: same tool, different arguments. Blocking that
// would stop a model correcting itself, which is the whole point of telling it
// what went wrong.
func TestRun_AllowsARetryWithDifferentArguments(t *testing.T) {
	t.Parallel()

	tool := queryTool("report_run", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(
			"report_run",
			map[string]any{"statuses": map[string]any{"item": []any{"A"}}},
		),
		runtimetest.ToolTurn("report_run", map[string]any{"statuses": []any{"A"}}),
		runtimetest.TextTurn("done"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_run"),
		Actor:      runtimetest.Actor(),
		Input:      "run the unbilled report",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, tool.Calls)
}

// A read that succeeded is not run again unchanged while nothing has written:
// it would return the record the turn already has. The model is pointed back
// to that result instead, and the turn goes on.
func TestRun_AnswersARepeatedReadFromTheFirst(t *testing.T) {
	t.Parallel()

	args := map[string]any{"status": "Completed"}
	tool := queryTool("shipment_list", map[string]any{"results": []any{}}, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_list", args),
		runtimetest.ToolTurn("shipment_list", args),
		runtimetest.TextTurn("nothing found"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_list"),
		Actor:      runtimetest.Actor(),
		Input:      "anything",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls, "the identical read is answered, not run")
	repeat := result.Messages[4]
	assert.False(t, repeat.ToolFailed, "a repeated read is not shown as a failure")
	assert.Contains(t, repeat.Content, "already ran earlier in this turn")
}
