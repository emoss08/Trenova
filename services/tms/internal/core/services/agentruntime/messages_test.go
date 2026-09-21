package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalRationale_UsesTheModelsOwnWordsWhenItGaveAny(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(rationaleInput{
		Narration: "  Reassigning to the Dallas terminal because the driver is out of hours.  ",
		ToolName:  "reassign_move",
		Arguments: map[string]any{"attemptSummary": "ignored while the model narrated"},
		Input:     "ignored while the model narrated",
	})

	assert.Equal(t, "Reassigning to the Dallas terminal because the driver is out of hours.", rationale)
}

func TestProposalRationale_SaysSoWhenTheModelExplainedNothing(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(rationaleInput{Narration: "   ", ToolName: "reassign_move"})

	assert.Contains(t, rationale, "reassign_move")
	assert.Contains(t, rationale, "without explaining why")
}

// A model that calls a tool without narrating usually put its reasoning in
// the call: raise_exception carries an attemptSummary, and most writes carry
// a reason or a note. The card said "without explaining why" above an
// argument that explained exactly why.
func TestProposalRationale_FallsBackToTheSummaryInTheArguments(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(rationaleInput{
		ToolName: "raise_exception",
		Arguments: map[string]any{
			"subjectId":      "shp_1",
			"attemptSummary": "  The shipment completed without a signed BOL.  ",
		},
		Input: "flag these for review",
	})

	assert.Equal(t, "The shipment completed without a signed BOL.", rationale)
}

func TestProposalRationale_PrefersTheMostSpecificArgument(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(rationaleInput{
		ToolName: "hold_shipment",
		Arguments: map[string]any{
			"note":   "a note",
			"reason": "Consignee is closed until Monday",
		},
	})

	assert.Equal(t, "Consignee is closed until Monday", rationale)
}

// With no narration and nothing in the call, the person's own request is the
// best account of why: they asked for it.
func TestProposalRationale_FallsBackToWhatThePersonAsked(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(rationaleInput{
		ToolName:  "raise_exception",
		Arguments: map[string]any{"subjectId": "shp_1", "reason": "", "note": 12},
		Input:     "  Flag SEED-DET-011 for review  ",
	})

	assert.Equal(
		t,
		"Asked to raise exception in reply to: “Flag SEED-DET-011 for review”",
		rationale,
	)
}

func TestProposalRationale_TruncatesTheFallbacksToo(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'b'
	}

	fromArgument := proposalRationale(rationaleInput{
		ToolName:  "hold_shipment",
		Arguments: map[string]any{"reason": string(long)},
	})
	fromInput := proposalRationale(rationaleInput{ToolName: "hold_shipment", Input: string(long)})

	assert.LessOrEqual(t, len([]rune(fromArgument)), maxRationaleChars+1)
	assert.LessOrEqual(t, len([]rune(fromInput)), maxRationaleChars+1)
}

func TestProposalRationale_TruncatesNarration(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'a'
	}

	rationale := proposalRationale(rationaleInput{Narration: string(long), ToolName: "reassign_move"})

	assert.LessOrEqual(t, len([]rune(rationale)), maxRationaleChars+1)
}

// The thread is replayed as its newest forty messages, and that cut lands
// wherever it lands — including between an assistant's tool calls and the
// results that answer them. A conversation opening on an orphaned tool result
// is rejected by every provider, so a long thread with heavy tool use began
// failing outright once it crossed the limit. History starts at a turn.
func TestToAdapterMessages_StartsAtAUserTurn(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleTool, ToolCallID: "call_0", ToolName: "get_shipment", Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "It is in Dallas."},
		{Role: conversation.RoleUser, Content: "And the driver?"},
		{Role: conversation.RoleAssistant, Content: "Sarah Williams."},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, 2)
	assert.Equal(t, serviceports.RoleUser, messages[0].Role)
	assert.Equal(t, "And the driver?", messages[0].Content)
}

func TestToAdapterMessages_KeepsAWholeHistoryThatAlreadyStartsRight(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "get_shipment"}}},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "get_shipment", Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "Dallas."},
	}

	assert.Len(t, toAdapterMessages(history, nil), 4)
}

// History replay carries the reasoning, since a provider that signs its
// thinking refuses a later tool result without it.
func TestToAdapterMessages_CarriesReasoning(t *testing.T) {
	t.Parallel()

	messages := toAdapterMessages([]conversation.Message{
		{Role: conversation.RoleUser, Content: "hold it"},
		{Role: conversation.RoleAssistant, Content: "Done.", Reasoning: &conversation.ReasoningTrace{Text: "t", Signature: "s"}},
	}, nil)

	require.Len(t, messages, 2)
	require.NotNil(t, messages[1].Reasoning)
	assert.Equal(t, "s", messages[1].Reasoning.Signature)
}

// A refused assistant turn is left out of the replay, and so must be the
// tool results that answered its calls: a result whose call is not in the
// conversation is an orphan every provider rejects.
func TestToAdapterMessages_DropsTheResultsOfARefusedTurnsCalls(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Delete every shipment."},
		{
			Role:      conversation.RoleAssistant,
			Refused:   true,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "search_shipments"}},
		},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "search_shipments", Content: "{}"},
		{Role: conversation.RoleUser, Content: "Fine, where is S1?"},
		{Role: conversation.RoleAssistant, Content: "Dallas."},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, 3)
	for _, msg := range messages {
		assert.NotEqual(t, serviceports.RoleTool, msg.Role)
	}
}

// Old tool results are the bulk of a long thread, and none of them is what
// the current question is about: a listing from twenty turns ago is stale by
// now and the model can fetch it again. Replay keeps them whole for the most
// recent turns and cuts the older ones down to their opening, so a long
// conversation keeps its words rather than losing its history to its data.
func TestToAdapterMessages_CompactsToolResultsFromOlderTurns(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("row,", 2000)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "List shipments."},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "list_shipments"}}},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "list_shipments", Content: big},
		{Role: conversation.RoleAssistant, Content: "Here they are."},
	}
	for turn := range recentToolTurns {
		id := fmt.Sprintf("c%d", turn+2)
		history = append(history,
			conversation.Message{Role: conversation.RoleUser, Content: "And again."},
			conversation.Message{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{ID: id, Name: "list_shipments"}}},
			conversation.Message{Role: conversation.RoleTool, ToolCallID: id, ToolName: "list_shipments", Content: big},
			conversation.Message{Role: conversation.RoleAssistant, Content: "Here they are."},
		)
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, len(history))
	oldest := messages[2]
	require.Equal(t, serviceports.RoleTool, oldest.Role)
	assert.Less(t, len(oldest.Content), len(big)/4)
	assert.True(t, strings.HasPrefix(oldest.Content, "row,row,"), "the opening is kept")
	assert.Contains(t, oldest.Content, "elided")
	assert.Contains(t, oldest.Content, "again", "the model is told it may fetch it again")

	newest := messages[len(messages)-2]
	require.Equal(t, serviceports.RoleTool, newest.Role)
	assert.Equal(t, big, newest.Content, "the recent turns keep their results whole")
}

// A short result is left alone whatever its age: the note would be longer
// than what it replaced.
func TestToAdapterMessages_LeavesShortOldResultsAlone(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "get_shipment"}}},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "get_shipment", Content: `{"status":"InTransit"}`},
		{Role: conversation.RoleAssistant, Content: "In transit."},
	}
	for turn := range recentToolTurns + 1 {
		history = append(history,
			conversation.Message{Role: conversation.RoleUser, Content: fmt.Sprintf("Turn %d", turn)},
			conversation.Message{Role: conversation.RoleAssistant, Content: "Noted."},
		)
	}

	messages := toAdapterMessages(history, nil)
	assert.Equal(t, `{"status":"InTransit"}`, messages[2].Content)
}

// A turn that died between issuing a tool call and recording its result
// leaves a call nothing answers, and a provider rejects the conversation
// for it as surely as for an orphaned result. The call is left out of the
// replay; the words around it are kept.
func TestToAdapterMessages_DropsCallsNothingAnswered(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{
			Role:    conversation.RoleAssistant,
			Content: "Looking.",
			ToolCalls: []conversation.ToolCallRecord{
				{ID: "c1", Name: "get_shipment"},
				{ID: "c2", Name: "get_worker"},
			},
		},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "get_shipment", Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "The run stopped before it finished."},
		{Role: conversation.RoleUser, Content: "Try again."},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{ID: "c3", Name: "get_shipment"}}},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, 5, "the assistant turn with nothing but an unanswered call is left out")
	require.Len(t, messages[1].ToolCalls, 1)
	assert.Equal(t, "c1", messages[1].ToolCalls[0].ID)
	assert.Equal(t, "Looking.", messages[1].Content)
	assert.Equal(t, serviceports.RoleTool, messages[2].Role)
	assert.Equal(t, serviceports.RoleUser, messages[4].Role)
}
