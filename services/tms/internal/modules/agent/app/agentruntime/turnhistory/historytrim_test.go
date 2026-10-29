package turnhistory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An older result is shortened, and it used to be cut through its fence: the
close tag went, and the note telling the model to fetch the result again sat
inside data it is told never to take instructions from. The shortened result
is a fence of its own, closed, with the note outside it.
*/
func TestReplayHistory_ShortensAnOldResultInsideAClosedFence(t *testing.T) {
	t.Parallel()

	stored := toolresult.Fence("shipment_list", strings.Repeat(`{"pro":"S1"},`, 400)) +
		"\n\nShown to the person as a table."
	history := append(
		runtimetest.ReadTurn("c1", "shipment_list", stored),
		runtimetest.QuietTurns(RecentToolTurns)...)

	messages, _ := replayHistory(history, nil)

	shortened := messages[2].Content
	require.Less(t, len(shortened), len(stored)/4)
	tool, payload, fenced := toolresult.Unfence(shortened)
	require.True(t, fenced, "the shortened result is still a closed fence")
	assert.Equal(t, "shipment_list", tool)
	assert.True(t, strings.HasPrefix(payload, `{"pro":"S1"}`), "its opening is kept")
	closeAt := strings.Index(shortened, toolresult.UntrustedCloseTag)
	note := shortened[closeAt:]
	assert.Contains(t, note, "shipment_list again", "the note, outside the fence, names the tool")
	assert.Contains(
		t,
		note,
		"Shown to the person as a table.",
		"a note that followed the result is kept",
	)
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
		calls = append(calls, conversation.ToolCallRecord{ID: id, Name: "shipment_list"})
		results = append(results, conversation.Message{
			Role:       conversation.RoleTool,
			ToolCallID: id,
			ToolName:   "shipment_list",
			Content:    toolresult.Fence("shipment_list", strings.Repeat("x", 11000)),
		})
	}
	history = append(
		history,
		conversation.Message{Role: conversation.RoleAssistant, ToolCalls: calls},
	)
	history = append(history, results...)

	messages, _ := replayHistory(history, nil)

	kept := 0
	for _, message := range messages {
		if message.Role == llmports.RoleTool {
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

	stored := toolresult.Fence("shipment_list", strings.Repeat("row,", 2000))
	history := runtimetest.ReadTurn("c1", "shipment_list", stored)
	for range RecentToolTurns + 1 {
		history = append(history, conversation.Message{
			Role:    conversation.RoleUser,
			Kind:    conversation.MessageKindDecisionNote,
			Content: "Approved.",
		})
	}

	messages, _ := replayHistory(history, nil)

	assert.Equal(t, stored, messages[2].Content)
}
