package turnhistory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toAdapterMessages(
	history []conversation.Message,
	outcomes []agentruntimeports.ProposalOutcome,
) []llmports.Message {
	messages, _ := replayHistory(history, outcomes)

	return messages
}

func TestProposalRationale_UsesTheModelsOwnWordsWhenItGaveAny(t *testing.T) {
	t.Parallel()

	rationale := ProposalRationale(RationaleInput{
		Narration: "  Reassigning to the Dallas terminal because the driver is out of hours.  ",
		ToolName:  "reassign_move",
		Arguments: map[string]any{"attemptSummary": "ignored while the model narrated"},
		Input:     "ignored while the model narrated",
	})

	assert.Equal(
		t,
		"Reassigning to the Dallas terminal because the driver is out of hours.",
		rationale,
	)
}

func TestProposalRationale_SaysSoWhenTheModelExplainedNothing(t *testing.T) {
	t.Parallel()

	rationale := ProposalRationale(RationaleInput{Narration: "   ", ToolName: "reassign_move"})

	assert.Contains(t, rationale, "reassign_move")
	assert.Contains(t, rationale, "without explaining why")
}

// A model that calls a tool without narrating usually put its reasoning in
// the call: agent_raise_exception carries an attemptSummary, and most writes carry
// a reason or a note. The card said "without explaining why" above an
// argument that explained exactly why.
func TestProposalRationale_FallsBackToTheSummaryInTheArguments(t *testing.T) {
	t.Parallel()

	rationale := ProposalRationale(RationaleInput{
		ToolName: "agent_raise_exception",
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

	rationale := ProposalRationale(RationaleInput{
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

	rationale := ProposalRationale(RationaleInput{
		ToolName:  "agent_raise_exception",
		Arguments: map[string]any{"subjectId": "shp_1", "reason": "", "note": 12},
		Input:     "  Flag SEED-DET-011 for review  ",
	})

	assert.Equal(
		t,
		"Asked to raise agent exception in reply to: “Flag SEED-DET-011 for review”",
		rationale,
	)
}

func TestProposalRationale_TruncatesTheFallbacksToo(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'b'
	}

	fromArgument := ProposalRationale(RationaleInput{
		ToolName:  "hold_shipment",
		Arguments: map[string]any{"reason": string(long)},
	})
	fromInput := ProposalRationale(RationaleInput{ToolName: "hold_shipment", Input: string(long)})

	assert.LessOrEqual(t, len([]rune(fromArgument)), maxRationaleChars+1)
	assert.LessOrEqual(t, len([]rune(fromInput)), maxRationaleChars+1)
}

func TestProposalRationale_TruncatesNarration(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'a'
	}

	rationale := ProposalRationale(
		RationaleInput{Narration: string(long), ToolName: "reassign_move"},
	)

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
		{
			Role:       conversation.RoleTool,
			ToolCallID: "call_0",
			ToolName:   "shipment_get",
			Content:    "{}",
		},
		{Role: conversation.RoleAssistant, Content: "It is in Dallas."},
		{Role: conversation.RoleUser, Content: "And the driver?"},
		{Role: conversation.RoleAssistant, Content: "Sarah Williams."},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, 2)
	assert.Equal(t, llmports.RoleUser, messages[0].Role)
	assert.Equal(t, "And the driver?", messages[0].Content)
}

func TestToAdapterMessages_KeepsAWholeHistoryThatAlreadyStartsRight(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_get"}},
		},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "shipment_get", Content: "{}"},
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
		{
			Role:      conversation.RoleAssistant,
			Content:   "Done.",
			Reasoning: &conversation.ReasoningTrace{Text: "t", Signature: "s"},
		},
	}, nil)

	require.Len(t, messages, 2)
	require.NotNil(t, messages[1].Reasoning)
	assert.Equal(t, "s", messages[1].Reasoning.Signature)
}

// History replay carries each reply's provider id, so the first call of a
// turn can name the last call of the turn before.
func TestToAdapterMessages_CarriesEachReplysProviderID(t *testing.T) {
	t.Parallel()

	provider := pulid.MustNew("aip_")
	messages := toAdapterMessages([]conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{
			Role:              conversation.RoleAssistant,
			Content:           "In Dallas.",
			ProviderID:        provider,
			ProviderMessageID: "msg_01ANSWER",
		},
	}, nil)

	require.Len(t, messages, 2)
	assert.Equal(t, "msg_01ANSWER", messages[1].ReplyID)
	assert.Equal(t, provider, messages[1].ReplyProvider)
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
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_list"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "shipment_list",
			Content:    "{}",
		},
		{Role: conversation.RoleUser, Content: "Fine, where is S1?"},
		{Role: conversation.RoleAssistant, Content: "Dallas."},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, 3)
	for _, msg := range messages {
		assert.NotEqual(t, llmports.RoleTool, msg.Role)
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
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_list"}},
		},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "shipment_list", Content: big},
		{Role: conversation.RoleAssistant, Content: "Here they are."},
	}
	for turn := range RecentToolTurns {
		id := fmt.Sprintf("c%d", turn+2)
		history = append(
			history,
			conversation.Message{Role: conversation.RoleUser, Content: "And again."},
			conversation.Message{
				Role:      conversation.RoleAssistant,
				ToolCalls: []conversation.ToolCallRecord{{ID: id, Name: "shipment_list"}},
			},
			conversation.Message{
				Role:       conversation.RoleTool,
				ToolCallID: id,
				ToolName:   "shipment_list",
				Content:    big,
			},
			conversation.Message{Role: conversation.RoleAssistant, Content: "Here they are."},
		)
	}

	messages := toAdapterMessages(history, nil)

	require.Len(t, messages, len(history))
	oldest := messages[2]
	require.Equal(t, llmports.RoleTool, oldest.Role)
	assert.Less(t, len(oldest.Content), len(big)/4)
	assert.True(t, strings.HasPrefix(oldest.Content, "row,row,"), "the opening is kept")
	assert.Contains(t, oldest.Content, "elided")
	assert.Contains(t, oldest.Content, "again", "the model is told it may fetch it again")

	newest := messages[len(messages)-2]
	require.Equal(t, llmports.RoleTool, newest.Role)
	assert.Equal(t, big, newest.Content, "the recent turns keep their results whole")
}

// A short result is left alone whatever its age: the note would be longer
// than what it replaced.
func TestToAdapterMessages_LeavesShortOldResultsAlone(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_get"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "shipment_get",
			Content:    `{"status":"InTransit"}`,
		},
		{Role: conversation.RoleAssistant, Content: "In transit."},
	}
	for turn := range RecentToolTurns + 1 {
		history = append(
			history,
			conversation.Message{
				Role:    conversation.RoleUser,
				Content: fmt.Sprintf("Turn %d", turn),
			},
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
				{ID: "c1", Name: "shipment_get"},
				{ID: "c2", Name: "worker_get"},
			},
		},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolName: "shipment_get", Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "The run stopped before it finished."},
		{Role: conversation.RoleUser, Content: "Try again."},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c3", Name: "shipment_get"}},
		},
	}

	messages := toAdapterMessages(history, nil)

	require.Len(
		t,
		messages,
		5,
		"the assistant turn with nothing but an unanswered call is left out",
	)
	require.Len(t, messages[1].ToolCalls, 1)
	assert.Equal(t, "c1", messages[1].ToolCalls[0].ID)
	assert.Equal(t, "Looking.", messages[1].Content)
	assert.Equal(t, llmports.RoleTool, messages[2].Role)
	assert.Equal(t, llmports.RoleUser, messages[4].Role)
}

// What a provider attached to a call is stored with the call and replayed
// with it, along with which provider it came from, so a later turn on the
// same provider can continue and one on another provider is not handed a
// field it does not understand.
func TestToolCallRecords_KeepTheProvidersOwnData(t *testing.T) {
	t.Parallel()

	providerID := pulid.MustNew("aip_")
	calls := []llmports.ToolCall{{
		ID:           "c1",
		Name:         "shipment_get",
		Arguments:    map[string]any{"id": "S1"},
		ProviderData: map[string]any{"google": map[string]any{"thought_signature": "sig"}},
		ProviderID:   providerID,
	}}

	records := ToToolCallRecords(calls)
	require.Len(t, records, 1)
	assert.Equal(t, calls[0].ProviderData, records[0].ProviderData)
	assert.Equal(t, providerID, records[0].ProviderID)

	replayed := fromToolCallRecords(records)
	require.Len(t, replayed, 1)
	assert.Equal(t, calls[0], replayed[0])
}
