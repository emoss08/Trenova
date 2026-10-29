package proposalledger

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Arguments that cannot be encoded are no one's duplicate.

Both sets encoded to the same empty key, so a second proposal whose arguments
also failed to encode was dropped as a copy of the first, whatever it asked
for.
*/
func TestPendingDuplicate_NeverMatchesArgumentsThatCannotBeEncoded(t *testing.T) {
	t.Parallel()

	call := llmports.ToolCall{
		ID:        "call_2",
		Name:      "report_update",
		Arguments: map[string]any{"definitionId": "rd_2", "cycle": make(chan int)},
	}
	earlier := []agentruntimeports.ProposalOutcome{{
		ToolName:   "report_update",
		Status:     agent.ProposalStatusPending,
		ToolParams: map[string]any{"definitionId": "rd_1", "cycle": make(chan int)},
	}}
	thisTurn := []agentruntimeports.PendingAction{{
		ToolName:  "report_update",
		Arguments: map[string]any{"definitionId": "rd_1", "cycle": make(chan int)},
	}}

	require.Empty(t, argumentsKey(call.Arguments), "the arguments must fail to encode")
	assert.False(t, PendingDuplicate(&call, nil, thisTurn),
		"a different proposal earlier in this turn is not this one")
	assert.False(t, PendingDuplicate(&call, earlier, nil),
		"a different proposal from an earlier turn is not this one")
	assert.False(t, PendingDuplicate(&call, earlier, thisTurn))
}

// A change the approver made before approving is part of what ran, and the
// model that proposed the original values is told, so its reply describes
// the message that was sent rather than the one it drafted.
func TestProposalOutcomeText_SaysWhatTheApproverChanged(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("worker_send_message", &agentruntimeports.ProposalOutcome{
		Status:        agent.ProposalStatusExecuted,
		Modifications: map[string]any{"priority": "high", "message": "Call dispatch now"},
	})

	assert.Contains(t, text, `after changing message = "Call dispatch now", priority = "high"`)
	assert.Contains(t, text, "ran successfully")

	plain := proposalOutcomeText(
		"worker_send_message",
		&agentruntimeports.ProposalOutcome{Status: agent.ProposalStatusExecuted},
	)
	assert.NotContains(t, plain, "after changing")
}
