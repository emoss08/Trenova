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

// A call that succeeded is not a failure to remember. Asking the same question
// twice in a turn is ordinary — a second lookup after a write, say.
func TestRun_DoesNotBlockRepeatingASuccessfulCall(t *testing.T) {
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

	_, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("list_shipments"),
		Actor:      testActor(),
		Input:      "anything",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, tool.Calls)
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
