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

func failedTurn() []conversation.Message {
	refusal := "Tool \"dispatch_rank_candidates\" was not run or proposed: its arguments do " +
		"not fit the tool.\n- shipmentId: required"

	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "Rank drivers for these loads."},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "r1", Name: "dispatch_rank_candidates"},
			{ID: "r2", Name: "dispatch_rank_candidates"},
			{ID: "r3", Name: "dispatch_rank_candidates"},
		}},
		{
			Role: conversation.RoleTool, ToolCallID: "r1", ToolName: "dispatch_rank_candidates",
			ToolFailed: true, Content: refusal,
		},
		{
			Role: conversation.RoleTool, ToolCallID: "r2", ToolName: "dispatch_rank_candidates",
			ToolFailed: true, Content: refusal,
		},
		{
			Role: conversation.RoleTool, ToolCallID: "r3", ToolName: "dispatch_rank_candidates",
			ToolFailed: true, Content: refusal,
		},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "a1", Name: "show_answer"},
		}},
		{
			Role: conversation.RoleTool, ToolCallID: "a1", ToolName: "show_answer",
			ToolFailed: true,
			Content:    "Tool \"show_answer\" was not run: answer is longer than 600 characters.",
		},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "a2", Name: "show_answer"},
		}},
		{
			Role: conversation.RoleTool, ToolCallID: "a2", ToolName: "show_answer",
			Content: "Shown beside the conversation.",
		},
		{Role: conversation.RoleAssistant, Content: "Here is the ranking."},
	}
}

func notices(messages []llmports.Message) []string {
	out := make([]string, 0, 2)
	for _, message := range messages {
		if message.Role == llmports.RoleSystem {
			out = append(out, message.Content)
		}
	}

	return out
}

/*
A turn made four calls that did not go through, the last a show_answer a
successful retry superseded. Asked on the next turn why four tools failed, the
model could name only three. Every earlier turn now ends with a notice naming
each failed or refused call and why, retried or not.
*/
func TestReplayHistory_NamesEveryFailedCallOfAnEarlierTurn(t *testing.T) {
	t.Parallel()

	history := append(failedTurn(), conversation.Message{
		Role: conversation.RoleUser, Content: "4 tools didn't go through, tell me why",
	})

	messages, _ := replayHistory(history, nil)

	found := notices(messages)
	require.Len(t, found, 1)
	notice := found[0]
	assert.Contains(t, notice, "4 tool calls in the turn above did not go through")
	assert.Contains(t, notice, "dispatch_rank_candidates (3 times): "+
		"its arguments do not fit the tool. - shipmentId: required")
	assert.Contains(t, notice, "show_answer: answer is longer than 600 characters.")
	assert.NotContains(t, notice, "was not run", "the reason is given without the tool's preamble")

	last := messages[len(messages)-1]
	assert.Equal(t, llmports.RoleUser, last.Role)
	assert.Equal(t, notice, messages[len(messages)-2].Content,
		"the notice closes the turn it describes, before the next question")
}

func TestReplayHistory_ClosesTheNewestTurnWithItsFailures(t *testing.T) {
	t.Parallel()

	messages, _ := replayHistory(failedTurn(), nil)

	last := messages[len(messages)-1]
	assert.Equal(t, llmports.RoleSystem, last.Role)
	assert.Contains(t, last.Content, "show_answer")
}

func TestReplayHistory_AddsNoNoticeForATurnWithoutFailures(t *testing.T) {
	t.Parallel()

	history := append(runtimetest.ReadTurn("c1", "shipment_list", "rows"),
		runtimetest.QuietTurns(2)...)

	messages, _ := replayHistory(history, nil)

	assert.Empty(t, notices(messages))
}

func TestReplayHistory_ReadsTheReasonOfAFencedFailure(t *testing.T) {
	t.Parallel()

	history := runtimetest.ReadTurn("c1", "shipment_get",
		toolresult.FenceFailure("shipment_get", "shipment not found"))
	history[2].ToolFailed = true

	messages, _ := replayHistory(history, nil)

	found := notices(messages)
	require.Len(t, found, 1)
	assert.Contains(t, found[0], "- shipment_get: shipment not found")
}

func TestReplayHistory_NamesACallNothingAnswered(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Look it up."},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c1", Name: "shipment_list"},
		}},
		{Role: conversation.RoleUser, Content: "Did it work?"},
	}

	messages, _ := replayHistory(history, nil)

	found := notices(messages)
	require.Len(t, found, 1)
	assert.Contains(t, found[0], "shipment_list: "+unansweredCallReason)
}

func TestReplayHistory_KeepsAFailureFromTheTurnTheWindowCutThrough(t *testing.T) {
	t.Parallel()

	cut := failedTurn()[5:]
	history := append(cut, conversation.Message{Role: conversation.RoleUser, Content: "Why?"})

	messages, _ := replayHistory(history, nil)

	require.NotEmpty(t, messages)
	assert.Equal(t, llmports.RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, turnOutOfView)
	assert.Contains(t, messages[0].Content, "show_answer: answer is longer than 600 characters.")
}

func TestReplayHistory_BoundsTheFailureNotice(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{{Role: conversation.RoleUser, Content: "Try them all."}}
	calls := make([]conversation.ToolCallRecord, 0, 40)
	results := make([]conversation.Message, 0, 40)
	for i := range 40 {
		id := fmt.Sprintf("c%d", i)
		calls = append(calls, conversation.ToolCallRecord{ID: id, Name: "shipment_get"})
		results = append(results, conversation.Message{
			Role: conversation.RoleTool, ToolCallID: id, ToolName: "shipment_get",
			ToolFailed: true,
			Content:    fmt.Sprintf("shipment %d not found %s", i, strings.Repeat("x", 500)),
		})
	}
	history = append(history,
		conversation.Message{Role: conversation.RoleAssistant, ToolCalls: calls})
	history = append(history, results...)

	messages, _ := replayHistory(history, nil)

	found := notices(messages)
	require.Len(t, found, 1)
	assert.Contains(t, found[0], "40 tool calls")
	assert.Contains(t, found[0], "and 32 more failed calls")
	assert.Less(t, len(found[0]), 2600)
}
