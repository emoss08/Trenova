package turnhistory

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A proposal's tool result said "awaiting review" forever, whatever the
// person had decided since, so the model re-proposed and misreported. The
// replay now carries each proposal's current state.
func TestToAdapterMessages_ReplaysWhatBecameOfEachProposal(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	executedAt := int64(1_790_000_000)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Fix the dates"},
		{
			ID:   sourceID,
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{
					ID:        "c1",
					Name:      "report_update",
					Arguments: map[string]any{"definitionId": "rd_1"},
				},
				{
					ID:        "c2",
					Name:      "report_update",
					Arguments: map[string]any{"definitionId": "rd_2"},
				},
				{ID: "c3", Name: "report_get_run", Arguments: map[string]any{"runId": "rrun_1"}},
			},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_update",
			Content:    "Recorded a proposal to run \"report_update\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c2",
			ToolName:   "report_update",
			Content:    "Recorded a proposal to run \"report_update\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c3",
			ToolName:   "report_get_run",
			Content:    "{\"status\":\"succeeded\"}",
		},
		{Role: conversation.RoleAssistant, Content: "Proposed two updates."},
	}
	outcomes := []agentruntimeports.ProposalOutcome{
		{
			SourceMessageID: sourceID,
			ToolName:        "report_update",
			Status:          agent.ProposalStatusExecutionFailed,
			ExecutionError:  "validation failed: unknown field \"\" on entity \"shipment_move\"",
		},
		{
			SourceMessageID: sourceID,
			ToolName:        "report_update",
			Status:          agent.ProposalStatusExecuted,
			ExecutedAt:      &executedAt,
		},
	}

	messages := toAdapterMessages(history, outcomes)
	require.Len(t, messages, 6)

	assert.Contains(t, messages[2].Content, "FAILED")
	assert.Contains(t, messages[2].Content, "unknown field")
	assert.Contains(t, messages[3].Content, "ran successfully")
	assert.Equal(
		t,
		"{\"status\":\"succeeded\"}",
		messages[4].Content,
		"a plain tool result is left alone",
	)
}

func TestToAdapterMessages_TellsTheModelAPendingProposalIsStillWaiting(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Copy it"},
		{
			ID:        sourceID,
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "report_fork"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_fork",
			Content:    "Recorded a proposal",
		},
	}
	outcomes := []agentruntimeports.ProposalOutcome{
		{SourceMessageID: sourceID, ToolName: "report_fork", Status: agent.ProposalStatusPending},
	}

	messages := toAdapterMessages(history, outcomes)
	require.Len(t, messages, 3)
	assert.Contains(t, messages[2].Content, "still waiting")
	assert.Contains(t, messages[2].Content, "Do not propose it again")
}

func TestToAdapterMessages_LeavesResultsAloneWithNoOutcomes(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Copy it"},
		{
			ID:        pulid.MustNew("amsg_"),
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "report_fork"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_fork",
			Content:    "Recorded a proposal",
		},
	}

	messages := toAdapterMessages(history, nil)
	assert.Equal(t, "Recorded a proposal", messages[2].Content)
}

// A proposal's result is replayed as what became of it. The rate its preview
// priced is still something the model read, and a reply quoting it a turn
// later was rewritten as if it had made the figure up.
func TestReplayHistory_KeepsTheFiguresOfAProposalsResultAsRead(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	filed := "Summary: The rate agreement would charge 5959.35 USD for the linehaul."
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Book it"},
		{
			ID:        sourceID,
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_create"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "shipment_create",
			Content:    filed,
		},
	}
	outcomes := []agentruntimeports.ProposalOutcome{{
		SourceMessageID: sourceID,
		ToolName:        "shipment_create",
		Status:          agent.ProposalStatusPending,
	}}

	messages, shortened := replayHistory(history, outcomes)

	require.Len(t, messages, 3)
	assert.NotContains(t, messages[2].Content, "5959.35")
	assert.Equal(t, []string{filed}, shortened)
}
