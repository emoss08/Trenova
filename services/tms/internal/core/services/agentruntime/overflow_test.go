package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readingTurn is a request that has read results, one call and its answer
// after another, each result of size characters, fenced as a tool's are.
func readingTurn(results, size int) *serviceports.ChatCompletionRequest {
	messages := []serviceports.Message{{Role: serviceports.RoleUser, Content: "Which loads are late?"}}
	for idx := range results {
		callID := fmt.Sprintf("call_%d", idx)
		messages = append(messages,
			serviceports.Message{
				Role:      serviceports.RoleAssistant,
				ToolCalls: []serviceports.ToolCall{{ID: callID, Name: "list_shipments"}},
			},
			serviceports.Message{
				Role:       serviceports.RoleTool,
				ToolCallID: callID,
				ToolName:   "list_shipments",
				Content:    FenceToolResult("list_shipments", strings.Repeat("r", size)),
			},
		)
	}

	return &serviceports.ChatCompletionRequest{System: "Help dispatch.", Messages: messages}
}

/*
An agent reads three long listings in a row. By the third call the request is
past what the window holds, so the oldest listing is cut to its opening, with a
note saying so, until the request fits. The listing the model is about to read
for the first time goes whole, and the turn's own copy of every listing is
untouched.
*/
func TestFitWindow_ShortensTheOldestResultsAndKeepsTheNewestWhole(t *testing.T) {
	t.Parallel()

	req := readingTurn(3, 10_000)
	original := req.Messages
	before := requestTokens(req)
	window := int(float64(before) * 0.9 / turnWindowShare)

	fitWindow(req, window)

	require.Len(t, req.Messages, len(original))
	oldest := req.Messages[2].Content
	assert.Contains(t, oldest, "was shortened to keep the request inside the model's context window")
	assert.Contains(t, oldest, "call list_shipments again")
	assert.Less(t, len(oldest), 2_500)
	assert.Equal(t, original[4].Content, req.Messages[4].Content,
		"one cut was enough, so the result after it is left as it was")
	assert.Equal(t, original[6].Content, req.Messages[6].Content,
		"the result the model has not answered yet is never cut")
	assert.LessOrEqual(t, requestTokens(req), int(float64(window)*turnWindowShare))

	assert.Len(t, original[2].Content, len(FenceToolResult("list_shipments", strings.Repeat("r", 10_000))),
		"the turn's own messages are copied, not cut")
	tool, payload, fenced := UnfenceToolResult(oldest)
	assert.True(t, fenced, "the shortened result is still fenced, with the note outside the fence")
	assert.Equal(t, "list_shipments", tool)
	assert.Len(t, payload, squeezedToolResultChars)
}

// A request that fits, or one whose window is not known yet, is sent as the
// turn holds it.
func TestFitWindow_LeavesARequestThatFits(t *testing.T) {
	t.Parallel()

	req := readingTurn(3, 10_000)
	messages := req.Messages

	fitWindow(req, 200_000)
	assert.Equal(t, messages, req.Messages)

	fitWindow(req, 0)
	assert.Equal(t, messages, req.Messages)
}

// When the window is so small that nothing older is left to cut, what can be
// cut is, and the newest result still goes whole: a request the provider may
// refuse is better than one that hides from the model what it just asked for.
func TestFitWindow_NeverCutsTheResultsTheModelHasNotRead(t *testing.T) {
	t.Parallel()

	req := readingTurn(2, 10_000)
	newest := req.Messages[4].Content

	fitWindow(req, 1_000)

	assert.Contains(t, req.Messages[2].Content, "was shortened")
	assert.Equal(t, newest, req.Messages[4].Content)
}

// A small result is not worth cutting: the note would free next to nothing.
func TestFitWindow_LeavesSmallResultsAlone(t *testing.T) {
	t.Parallel()

	req := readingTurn(3, 600)
	small := req.Messages[2].Content

	fitWindow(req, 1_000)

	assert.Equal(t, small, req.Messages[2].Content)
}

/*
A turn on a provider configured with a small window reads three large records.
From the third call on, the request carries the earlier reads shortened, while
what the turn hands back to be saved keeps every read whole.
*/
func TestRun_ShortensEarlierReadsWhenTheTurnOutgrowsTheWindow(t *testing.T) {
	t.Parallel()

	tool := queryTool("get_shipment", map[string]any{"notes": strings.Repeat("n", 9_000)}, nil)
	read := func(id string) *serviceports.ChatCompletionResult {
		turn := toolTurn("get_shipment", map[string]any{"shipmentId": id})
		turn.ContextWindow = 4_096

		return turn
	}
	answer := textTurn("Those loads are running late.")
	answer.ContextWindow = 4_096
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		read("shp_1"), read("shp_2"), read("shp_3"), answer,
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "Which of these are late?",
	})
	require.NoError(t, err)
	require.Len(t, completion.Requests, 4)
	assert.Equal(t, 4_096, result.ContextWindow, "the configured window travels with the result")

	last := completion.Requests[3].Messages
	var results []string
	for _, message := range last {
		if message.Role == serviceports.RoleTool {
			results = append(results, message.Content)
		}
	}
	require.Len(t, results, 3)
	assert.Contains(t, results[0], "was shortened")
	assert.Contains(t, results[1], "was shortened")
	assert.NotContains(t, results[2], "was shortened", "the newest read goes whole")

	for _, message := range result.Messages {
		if message.ToolName == "get_shipment" {
			assert.NotContains(t, message.Content, "was shortened", "what is saved is never cut")
		}
	}
}

// Before any model has answered, the turn knows no window and cuts nothing;
// after, a configured window wins over the one the model id names.
func TestTurnWindow_IsTheAnsweringProvidersWindow(t *testing.T) {
	t.Parallel()

	turn := &Turn{result: &serviceports.RunResult{}}
	assert.Zero(t, turn.window())

	turn.result.Model = "claude-sonnet-4-5"
	assert.Equal(t, 200_000, turn.window())

	turn.result.ContextWindow = 64_000
	assert.Equal(t, 64_000, turn.window())
}
