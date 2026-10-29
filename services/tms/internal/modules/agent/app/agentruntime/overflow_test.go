package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A turn on a provider configured with a small window reads three large records.
From the third call on, the request carries the earlier reads shortened, while
what the turn hands back to be saved keeps every read whole.
*/
func TestRun_ShortensEarlierReadsWhenTheTurnOutgrowsTheWindow(t *testing.T) {
	t.Parallel()

	tool := queryTool("shipment_get", map[string]any{"notes": strings.Repeat("n", 9_000)}, nil)
	read := func(id string) *llmports.ChatCompletionResult {
		turn := runtimetest.ToolTurn("shipment_get", map[string]any{"shipmentId": id})
		turn.ContextWindow = 4_096

		return turn
	}
	answer := runtimetest.TextTurn("Those loads are running late.")
	answer.ContextWindow = 4_096
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		read("shp_1"), read("shp_2"), read("shp_3"), answer,
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Which of these are late?",
	})
	require.NoError(t, err)
	require.Len(t, completion.Requests, 4)
	assert.Equal(t, 4_096, result.ContextWindow, "the configured window travels with the result")

	last := completion.Requests[3].Messages
	var results []string
	for _, message := range last {
		if message.Role == llmports.RoleTool {
			results = append(results, message.Content)
		}
	}
	require.Len(t, results, 3)
	assert.Contains(t, results[0], "was shortened")
	assert.Contains(t, results[1], "was shortened")
	assert.NotContains(t, results[2], "was shortened", "the newest read goes whole")

	for _, message := range result.Messages {
		if message.ToolName == "shipment_get" {
			assert.NotContains(t, message.Content, "was shortened", "what is saved is never cut")
		}
	}
}

// Before any model has answered, the turn knows no window and cuts nothing;
// after, a configured window wins over the one the model id names.
func TestTurnWindow_IsTheAnsweringProvidersWindow(t *testing.T) {
	t.Parallel()

	turn := &Turn{result: &agentruntimeports.RunResult{}}
	assert.Zero(t, turn.window())

	turn.result.Model = "claude-sonnet-4-5"
	assert.Equal(t, 200_000, turn.window())

	turn.result.ContextWindow = 64_000
	assert.Equal(t, 64_000, turn.window())
}
