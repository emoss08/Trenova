package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/stretchr/testify/assert"
)

// A result is only spoken of once the write ran. An approval still being
// carried out has made nothing yet.
func TestDecisionNote_SaysNothingMadeBeforeTheWriteRuns(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ToolName: "report_create",
		Status:   agent.ProposalStatusAccepted,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			IDs:    map[string]string{"definitionId": "rd_01"},
		},
	}

	assert.Equal(t, "Approved report_create; it is being carried out.", decisionLine(proposal))
	assert.Empty(t, producedNote(proposal))
}

// A tool that names nothing leaves the note as it was.
func TestDecisionNote_KeepsThePlainLineWithoutAResult(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ToolName: "report_create_dashboard",
		Status:   agent.ProposalStatusExecuted,
	}

	assert.Equal(t, "Approved report_create_dashboard, and it ran.", decisionLine(proposal))
	assert.Empty(t, producedNote(proposal))
}

func TestDecisionHeadline_IsTheFirstLineOnly(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Approved report_create, and it ran.",
		DecisionHeadline("  Approved report_create, and it ran.\nDecision on proposal ap_1."))
	assert.Equal(
		t,
		"Rejected dispatch_assign_move.",
		DecisionHeadline("Rejected dispatch_assign_move."),
	)
	assert.Empty(t, DecisionHeadline(" \n "))
}
