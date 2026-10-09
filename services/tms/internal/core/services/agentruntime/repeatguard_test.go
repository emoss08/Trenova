package agentruntime

import (
	"errors"
	"strings"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
	tool := queryTool("run_report", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("run_report", args),
		toolTurn("run_report", args),
		textTurn("I could not start it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("run_report"),
		Actor:      testActor(),
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
	tool := queryTool("run_report", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("run_report", args),
		toolTurn("run_report", args),
		toolTurn("run_report", args),
		textTurn("The report refused the status filter, so it did not run."),
		toolTurn("run_report", args),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("run_report"),
		Actor:      testActor(),
		Input:      "run the unbilled report",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls)
	assert.Equal(t, 3, result.ToolCallsUsed, "the turn stopped well short of its budget")
	assert.Equal(t, "The report refused the status filter, so it did not run.", result.Reply)
	require.Len(t, completion.Requests, 4)
	final := completion.Requests[3]
	assert.Empty(t, final.Tools, "the answer is asked for with no tools")
	assert.Equal(t, stuckNote, final.Messages[len(final.Messages)-1].Content)
}

// The retry that was wanted: same tool, different arguments. Blocking that
// would stop a model correcting itself, which is the whole point of telling it
// what went wrong.
func TestRun_AllowsARetryWithDifferentArguments(t *testing.T) {
	t.Parallel()

	tool := queryTool("run_report", nil, errors.New("expected a list, got map"))
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("run_report", map[string]any{"statuses": map[string]any{"item": []any{"A"}}}),
		toolTurn("run_report", map[string]any{"statuses": []any{"A"}}),
		textTurn("done"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	_, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("run_report"),
		Actor:      testActor(),
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
	tool := queryTool("list_shipments", map[string]any{"results": []any{}}, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("list_shipments", args),
		toolTurn("list_shipments", args),
		textTurn("nothing found"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("list_shipments"),
		Actor:      testActor(),
		Input:      "anything",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls, "the identical read is answered, not run")
	repeat := result.Messages[4]
	assert.False(t, repeat.ToolFailed, "a repeated read is not shown as a failure")
	assert.Contains(t, repeat.Content, "already ran earlier in this turn")
}

// Key order is a serialisation detail of whichever provider produced the call,
// not a difference in what was asked.
func TestCallKey_IgnoresArgumentOrder(t *testing.T) {
	t.Parallel()

	first, ok := callKey(serviceports.ToolCall{
		Name:      "run_report",
		Arguments: map[string]any{"a": 1, "b": 2},
	})
	require.True(t, ok)
	second, ok := callKey(serviceports.ToolCall{
		Name:      "run_report",
		Arguments: map[string]any{"b": 2, "a": 1},
	})
	require.True(t, ok)

	assert.Equal(t, first, second)
}

func TestCallKey_SeparatesToolsThatShareArguments(t *testing.T) {
	t.Parallel()

	args := map[string]any{"id": "shp_1"}
	first, _ := callKey(serviceports.ToolCall{Name: "get_shipment", Arguments: args})
	second, _ := callKey(serviceports.ToolCall{Name: "cancel_shipment", Arguments: args})

	assert.NotEqual(t, first, second)
}

func TestFanOutNote_RemindsOnlyFromTheThirdSingleFetch(t *testing.T) {
	t.Parallel()

	assert.Empty(t, fanOutNote("get_billing_queue_item", 2))
	assert.Empty(t, fanOutNote("list_billing_queue_items", 5))
	assert.Contains(t, fanOutNote("get_billing_queue_item", 3), "3rd get_billing_queue_item call")
	assert.Contains(t, fanOutNote("get_billing_queue_item", 12), "12th")
}

func TestRepeatGuard_AnswersARepeatedReadUntilSomethingWrites(t *testing.T) {
	t.Parallel()

	guard := newRepeatGuard()
	read := serviceports.ToolCall{Name: "get_billing_queue_item", Arguments: map[string]any{"billingQueueItemId": "bqi_1"}}
	other := serviceports.ToolCall{Name: "get_billing_queue_item", Arguments: map[string]any{"billingQueueItemId": "bqi_2"}}
	write := serviceports.ToolCall{Name: "approve_billing_queue_item", Arguments: map[string]any{"billingQueueItemId": "bqi_1"}}

	assert.False(t, guard.readBefore(read), "a first read runs")
	guard.ran(read)
	assert.True(t, guard.readBefore(read), "the same read again is answered from the first")
	assert.False(t, guard.readBefore(other), "a read with other arguments runs")

	guard.ran(write)
	assert.False(t, guard.readBefore(read), "after a write the record may have changed, so it is read again")
	assert.False(t, guard.readBefore(write), "a write is never answered from an earlier one")
}
