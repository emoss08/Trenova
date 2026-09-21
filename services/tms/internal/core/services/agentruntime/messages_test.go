package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalRationale_UsesTheModelsOwnWordsWhenItGaveAny(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(
		"  Reassigning to the Dallas terminal because the driver is out of hours.  ",
		"reassign_move",
	)

	assert.Equal(t, "Reassigning to the Dallas terminal because the driver is out of hours.", rationale)
}

func TestProposalRationale_SaysSoWhenTheModelExplainedNothing(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale("   ", "reassign_move")

	assert.Contains(t, rationale, "reassign_move")
	assert.Contains(t, rationale, "without explaining why")
}

func TestProposalRationale_TruncatesNarration(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'a'
	}

	rationale := proposalRationale(string(long), "reassign_move")

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

	messages := toAdapterMessages(history)

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

	assert.Len(t, toAdapterMessages(history), 4)
}

// History replay carries the reasoning, since a provider that signs its
// thinking refuses a later tool result without it.
func TestToAdapterMessages_CarriesReasoning(t *testing.T) {
	t.Parallel()

	messages := toAdapterMessages([]conversation.Message{
		{Role: conversation.RoleUser, Content: "hold it"},
		{Role: conversation.RoleAssistant, Content: "Done.", Reasoning: &conversation.ReasoningTrace{Text: "t", Signature: "s"}},
	})

	require.Len(t, messages, 2)
	require.NotNil(t, messages[1].Reasoning)
	assert.Equal(t, "s", messages[1].Reasoning.Signature)
}
