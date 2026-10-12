package agentflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

// rendezvousWait is how long a read waits for the other to start. Run one
// after another, the first never sees the second and gives up.
const rendezvousWait = 5 * time.Second

/*
Two reads in one reply run as two activities at once: each waits for the
other to start before it answers. The first is also held until the second
has finished, and the transcript still reads in the order the model asked.
*/
func TestRun_ReadsInOneReplyRunAtOnceAndSettleInOrder(t *testing.T) {
	t.Parallel()

	var (
		arrived      sync.WaitGroup
		secondDone   = make(chan struct{})
		mu           sync.Mutex
		finishOrder  []string
		bothArrived  = make(chan struct{})
		arrivedClose sync.Once
	)
	arrived.Add(2)
	go func() {
		arrived.Wait()
		arrivedClose.Do(func() { close(bothArrived) })
	}()

	workers := &agenttest.StubQueryTool{
		ToolName: "list_workers",
		Result:   map[string]any{"results": []any{"Maria"}},
	}
	customers := &agenttest.StubQueryTool{
		ToolName: "search_customer",
		Result:   map[string]any{"results": []any{"Acme"}},
	}
	h := newHarness(t, harnessParams{
		query: []agentports.AgentQueryTool{workers, customers},
		wrapTool: func(
			run func(context.Context, converter.EncodedValues) (*ToolResult, error),
		) func(context.Context, converter.EncodedValues) (*ToolResult, error) {
			return func(ctx context.Context, input converter.EncodedValues) (*ToolResult, error) {
				name := activity.GetInfo(ctx).ActivityType.Name
				arrived.Done()
				select {
				case <-bothArrived:
				case <-time.After(rendezvousWait):
					return nil, errors.New(name + " ran without the other read")
				}
				if name == "list_workers" {
					<-secondDone
				}
				result, err := run(ctx, input)
				mu.Lock()
				finishOrder = append(finishOrder, name)
				mu.Unlock()
				if name == "search_customer" {
					close(secondDone)
				}

				return result, err
			}
		},
	})
	h.replies(
		&agentports.ChatCompletionResult{
			ToolCalls: []agentports.ToolCall{
				{ID: "call_1", Name: "list_workers", Arguments: map[string]any{"status": "Active"}},
				{ID: "call_2", Name: "search_customer", Arguments: map[string]any{"query": "Acme"}},
			},
			ModelIdentifier: "test-model",
		},
		textReply("Maria is free, and Acme is a customer."),
	)

	result := h.run(t, runContext("list_workers", "search_customer"))

	require.Empty(t, result.Err)
	assert.Equal(t, []string{"search_customer", "list_workers"}, finishOrder,
		"the second read finished first")
	assert.Equal(t, 1, workers.Calls)
	assert.Equal(t, 1, customers.Calls)
	assert.Equal(t, 2, result.Outcome.Result.ToolCallsUsed)

	var tools []conversation.Message
	for _, message := range result.Outcome.Result.Messages {
		if message.Role == conversation.RoleTool {
			tools = append(tools, message)
		}
	}
	require.Len(t, tools, 2)
	assert.Equal(t, "call_1", tools[0].ToolCallID, "results settle in the order they were asked")
	assert.Contains(t, tools[0].Content, "Maria")
	assert.Equal(t, "call_2", tools[1].ToolCallID)
	assert.Contains(t, tools[1].Content, "Acme")
	assert.False(t, tools[0].ToolFailed)
	assert.False(t, tools[1].ToolFailed)
}

/*
An execution started before reads ran together answers the change with the
default version, and its reads run as they did then: the second activity is
scheduled only once the first has finished.
*/
func TestRun_AnExecutionFromBeforeTheChangeRunsItsReadsInTurn(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		steps []string
	)
	note := func(step string) {
		mu.Lock()
		steps = append(steps, step)
		mu.Unlock()
	}

	h := newHarness(t, harnessParams{
		query: []agentports.AgentQueryTool{
			&agenttest.StubQueryTool{ToolName: "list_workers", Result: map[string]any{}},
			&agenttest.StubQueryTool{ToolName: "search_customer", Result: map[string]any{}},
		},
		wrapTool: func(
			run func(context.Context, converter.EncodedValues) (*ToolResult, error),
		) func(context.Context, converter.EncodedValues) (*ToolResult, error) {
			return func(ctx context.Context, input converter.EncodedValues) (*ToolResult, error) {
				name := activity.GetInfo(ctx).ActivityType.Name
				note("start " + name)
				result, err := run(ctx, input)
				note("end " + name)

				return result, err
			}
		},
	})
	h.env.OnGetVersion("agent-loop-concurrent-reads", workflow.DefaultVersion, 1).
		Return(workflow.DefaultVersion)
	h.replies(
		&agentports.ChatCompletionResult{
			ToolCalls: []agentports.ToolCall{
				{ID: "call_1", Name: "list_workers", Arguments: map[string]any{"status": "Active"}},
				{ID: "call_2", Name: "search_customer", Arguments: map[string]any{"query": "Acme"}},
			},
			ModelIdentifier: "test-model",
		},
		textReply("Done."),
	)

	result := h.run(t, runContext("list_workers", "search_customer"))

	require.Empty(t, result.Err)
	assert.Equal(t, []string{
		"start list_workers", "end list_workers", "start search_customer", "end search_customer",
	}, steps)
}
