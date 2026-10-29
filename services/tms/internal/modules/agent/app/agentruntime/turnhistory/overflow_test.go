package turnhistory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readingTurn is a request that has read results, one call and its answer
// after another, each result of size characters, fenced as a tool's are.
func readingTurn(results, size int) *llmports.ChatCompletionRequest {
	messages := []llmports.Message{{Role: llmports.RoleUser, Content: "Which loads are late?"}}
	for idx := range results {
		callID := fmt.Sprintf("call_%d", idx)
		messages = append(messages,
			llmports.Message{
				Role:      llmports.RoleAssistant,
				ToolCalls: []llmports.ToolCall{{ID: callID, Name: "shipment_list"}},
			},
			llmports.Message{
				Role:       llmports.RoleTool,
				ToolCallID: callID,
				ToolName:   "shipment_list",
				Content:    toolresult.Fence("shipment_list", strings.Repeat("r", size)),
			},
		)
	}

	return &llmports.ChatCompletionRequest{System: "Help dispatch.", Messages: messages}
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

	FitWindow(req, window)

	require.Len(t, req.Messages, len(original))
	oldest := req.Messages[2].Content
	assert.Contains(
		t,
		oldest,
		"was shortened to keep the request inside the model's context window",
	)
	assert.Contains(t, oldest, "call shipment_list again")
	assert.Less(t, len(oldest), 2_500)
	assert.Equal(t, original[4].Content, req.Messages[4].Content,
		"one cut was enough, so the result after it is left as it was")
	assert.Equal(t, original[6].Content, req.Messages[6].Content,
		"the result the model has not answered yet is never cut")
	assert.LessOrEqual(t, requestTokens(req), int(float64(window)*turnWindowShare))

	assert.Len(
		t,
		original[2].Content,
		len(toolresult.Fence("shipment_list", strings.Repeat("r", 10_000))),
		"the turn's own messages are copied, not cut",
	)
	tool, payload, fenced := toolresult.Unfence(oldest)
	assert.True(t, fenced, "the shortened result is still fenced, with the note outside the fence")
	assert.Equal(t, "shipment_list", tool)
	assert.Len(t, payload, squeezedToolResultChars)
}

// A request that fits, or one whose window is not known yet, is sent as the
// turn holds it.
func TestFitWindow_LeavesARequestThatFits(t *testing.T) {
	t.Parallel()

	req := readingTurn(3, 10_000)
	messages := req.Messages

	FitWindow(req, 200_000)
	assert.Equal(t, messages, req.Messages)

	FitWindow(req, 0)
	assert.Equal(t, messages, req.Messages)
}

// When the window is so small that nothing older is left to cut, what can be
// cut is, and the newest result still goes whole: a request the provider may
// refuse is better than one that hides from the model what it just asked for.
func TestFitWindow_NeverCutsTheResultsTheModelHasNotRead(t *testing.T) {
	t.Parallel()

	req := readingTurn(2, 10_000)
	newest := req.Messages[4].Content

	FitWindow(req, 1_000)

	assert.Contains(t, req.Messages[2].Content, "was shortened")
	assert.Equal(t, newest, req.Messages[4].Content)
}

// A small result is not worth cutting: the note would free next to nothing.
func TestFitWindow_LeavesSmallResultsAlone(t *testing.T) {
	t.Parallel()

	req := readingTurn(3, 600)
	small := req.Messages[2].Content

	FitWindow(req, 1_000)

	assert.Equal(t, small, req.Messages[2].Content)
}
