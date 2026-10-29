package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestExecutionFailureReason_KeepsTheListedProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "one line with wrapped causes after it",
			text: "tiles[0].definitionId: This report is not available.\nwrapped: cause",
			want: "tiles[0].definitionId: This report is not available.",
		},
		{
			name: "a header and its problems",
			text: "validation failed:\n- bol: taken\n- weight: too heavy",
			want: "validation failed: bol: taken; weight: too heavy",
		},
		{
			name: "no more than five problems",
			text: "validation failed:\n- a: 1\n- b: 2\n- c: 3\n- d: 4\n- e: 5\n- f: 6\n- g: 7",
			want: "validation failed: a: 1; b: 2; c: 3; d: 4; e: 5; and 2 more",
		},
		{
			name: "problems end at the first line that is not one",
			text: "validation failed:\n- a: 1\ncaused by: x\n- b: 2",
			want: "validation failed: a: 1",
		},
		{
			name: "nothing",
			text: "  ",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, executionFailureReason(tt.text))
		})
	}
}

func TestFollowUpInstruction_KeepsTheReplyToWhatTheNoteAndTheCardHold(t *testing.T) {
	t.Parallel()

	assert.Contains(t, FollowUpInstruction,
		"Report only what this note and the proposal's card hold")
	assert.Contains(t, FollowUpInstruction, "Do not propose the same change again.")
	assert.Contains(t, FollowUpInstruction, continueTheWork,
		"an approved step toward the person's request is followed by the next one")
	assert.Contains(t, notedInstruction, continueTheWork)
}

// "Approved shipment_update, and it ran" left the agent nothing to say, and
// gpt-6-luna reported the weight change without the weight. A write that
// reports no result says what it ran with, the approver's changes winning.
func TestRanWith_NamesTheApprovedValuesOfAWriteWithNoResult(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		Status:   agent.ProposalStatusExecuted,
		ToolName: "shipment_update",
		ToolParams: map[string]any{
			"shipmentId": "shp_1",
			"weight":     float64(38500),
			"pieces":     float64(10),
		},
	}
	changed := []*agentdecision.AgentDecision{{
		ProposalID:    &proposal.ID,
		Modifications: map[string]any{"pieces": float64(12)},
	}}

	got := ranWith(proposal, changed)
	assert.Equal(t, " It ran with pieces: 12; weight: 38500.", got)

	proposal.ExecutionResult = &agentrecord.ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   "On-time",
	}
	assert.Empty(
		t,
		ranWith(proposal, nil),
		"a write that describes its own result says that instead",
	)

	proposal.ExecutionResult = nil
	proposal.Status = agent.ProposalStatusRejected
	assert.Empty(t, ranWith(proposal, nil))
}
