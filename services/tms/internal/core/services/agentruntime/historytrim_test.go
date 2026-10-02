package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readTurn(id, tool, content string) []conversation.Message {
	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "Look it up."},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: id, Name: tool}},
		},
		{Role: conversation.RoleTool, ToolCallID: id, ToolName: tool, Content: content},
		{Role: conversation.RoleAssistant, Content: "Here it is."},
	}
}

func quietTurns(n int) []conversation.Message {
	out := make([]conversation.Message, 0, 2*n)
	for i := range n {
		out = append(out,
			conversation.Message{Role: conversation.RoleUser, Content: fmt.Sprintf("Turn %d", i)},
			conversation.Message{Role: conversation.RoleAssistant, Content: "Noted."},
		)
	}

	return out
}

/*
An older result is shortened, and it used to be cut through its fence: the
close tag went, and the note telling the model to fetch the result again sat
inside data it is told never to take instructions from. The shortened result
is a fence of its own, closed, with the note outside it.
*/
func TestReplayHistory_ShortensAnOldResultInsideAClosedFence(t *testing.T) {
	t.Parallel()

	stored := FenceToolResult("list_shipments", strings.Repeat(`{"pro":"S1"},`, 400)) +
		"\n\nShown to the person as a table."
	history := append(readTurn("c1", "list_shipments", stored), quietTurns(recentToolTurns)...)

	messages, _ := replayHistory(history, nil)

	shortened := messages[2].Content
	require.Less(t, len(shortened), len(stored)/4)
	tool, payload, fenced := UnfenceToolResult(shortened)
	require.True(t, fenced, "the shortened result is still a closed fence")
	assert.Equal(t, "list_shipments", tool)
	assert.True(t, strings.HasPrefix(payload, `{"pro":"S1"}`), "its opening is kept")
	closeAt := strings.Index(shortened, untrustedCloseTag)
	note := shortened[closeAt:]
	assert.Contains(t, note, "list_shipments again", "the note, outside the fence, names the tool")
	assert.Contains(t, note, "Shown to the person as a table.", "a note that followed the result is kept")
	assert.NotContains(t, shortened[:closeAt], "again")
}

/*
Three recent turns were kept whole however large they were, so a turn that
read a dozen long listings resent all of them on every call after it. The
recent results are kept whole up to a size, newest first; past it the older
ones are shortened like any other.
*/
func TestReplayHistory_KeepsRecentResultsWholeOnlyUpToABudget(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{{Role: conversation.RoleUser, Content: "List everything."}}
	calls := make([]conversation.ToolCallRecord, 0, 6)
	results := make([]conversation.Message, 0, 6)
	for i := range 6 {
		id := fmt.Sprintf("c%d", i)
		calls = append(calls, conversation.ToolCallRecord{ID: id, Name: "list_shipments"})
		results = append(results, conversation.Message{
			Role:       conversation.RoleTool,
			ToolCallID: id,
			ToolName:   "list_shipments",
			Content:    FenceToolResult("list_shipments", strings.Repeat("x", 11000)),
		})
	}
	history = append(history, conversation.Message{Role: conversation.RoleAssistant, ToolCalls: calls})
	history = append(history, results...)

	messages, _ := replayHistory(history, nil)

	kept := 0
	for _, message := range messages {
		if message.Role == serviceports.RoleTool {
			kept += len(message.Content)
		}
	}
	assert.LessOrEqual(t, kept, wholeToolResultBudget+2*compactedToolResultChars*6)
	assert.Equal(t, results[5].Content, messages[len(messages)-1].Content, "the newest stays whole")
	assert.Less(t, len(messages[2].Content), len(results[0].Content), "the oldest is shortened")
}

// A decision note is the person's answer arriving as a message, not a turn of
// questions, so it does not push results out of the recent turns.
func TestReplayHistory_DoesNotCountDecisionNotesAsTurns(t *testing.T) {
	t.Parallel()

	stored := FenceToolResult("list_shipments", strings.Repeat("row,", 2000))
	history := readTurn("c1", "list_shipments", stored)
	for range recentToolTurns + 1 {
		history = append(history, conversation.Message{
			Role:    conversation.RoleUser,
			Kind:    conversation.MessageKindDecisionNote,
			Content: "Approved.",
		})
	}

	messages, _ := replayHistory(history, nil)

	assert.Equal(t, stored, messages[2].Content)
}

/*
The grounding guard checks the figures in a reply against what the turn read.
A figure the model quotes from a result an earlier turn read is supported, and
shortening that result in the replay must not make it look invented.
*/
func TestOpenTurn_KeepsTheFiguresOfShortenedResultsAsEvidence(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat(`{"pro":"S1"},`, 100) + `{"pro":"S9","rate":"4817.25"}`
	history := append(
		readTurn("c1", "list_shipments", FenceToolResult("list_shipments", payload)),
		quietTurns(recentToolTurns)...,
	)
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		History:    history,
		Input:      "What was S9's rate?",
	}

	turn := rt.OpenTurn(t.Context(), req)
	require.NotContains(t, strings.Join(turn.groundingTexts(), "\n"), "4817.25",
		"the replay no longer carries the figure")

	want := decimal.RequireFromString("4817.25")
	assert.True(t, containsDecimal(turn.groundingSupport(), want))

	restored := rt.RestoreTurn(req, turn.State())
	assert.True(t, containsDecimal(restored.groundingSupport(), want), "workflow turns keep it too")
}

func containsDecimal(values []decimal.Decimal, want decimal.Decimal) bool {
	for _, value := range values {
		if value.Equal(want) {
			return true
		}
	}

	return false
}
