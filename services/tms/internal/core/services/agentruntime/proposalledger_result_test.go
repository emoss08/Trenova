package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

/*
A later turn reads what an approved write made, not only that it ran.

After an approved create_report the model knew a report had been saved and
not which one, and passed the proposal's id to describe_report. Every later
turn now reads the report's name and the id to use.
*/
func TestProposalOutcomeText_NamesWhatAnExecutedWriteMade(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("create_report", serviceports.ProposalOutcome{
		Status: agent.ProposalStatusExecuted,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			Name:   "Shipments for Peak Distributing",
			IDs:    map[string]string{"definitionId": "rd_01JPEAK"},
		},
	})

	assert.Equal(t,
		`The person approved the proposal to run "create_report" and it ran successfully. `+
			`It created the report "Shipments for Peak Distributing". It produced definitionId `+
			`rd_01JPEAK; use that id, not the proposal's, when you refer to what it made. `+
			`The change has been made; do not propose it again.`,
		text,
	)
}

func TestProposalOutcomeText_ReadsAsBeforeWithoutAResult(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("create_dashboard", serviceports.ProposalOutcome{
		Status: agent.ProposalStatusExecuted,
	})

	assert.Equal(t,
		`The person approved the proposal to run "create_dashboard" and it ran successfully. `+
			`The change has been made; do not propose it again.`,
		text,
	)
}

// A write that ran on its own was not approved by anyone, and a later turn
// must not tell the person they approved it. A private report is saved this
// way.
func TestProposalOutcomeText_DoesNotCallAnAutomaticWriteApproved(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("create_report", serviceports.ProposalOutcome{
		Status:       agent.ProposalStatusExecuted,
		AutonomyTier: agent.TierAutoExecute,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			Name:   "Lanes",
			IDs:    map[string]string{"definitionId": "rd_01"},
		},
	})

	assert.Equal(t,
		`The call to "create_report" ran on its own, without needing approval, and succeeded. `+
			`It created the report "Lanes". It produced definitionId rd_01; use that id, not `+
			`the proposal's, when you refer to what it made. The change has been made; do not `+
			`make it again.`,
		text,
	)

	failed := proposalOutcomeText("create_report", serviceports.ProposalOutcome{
		Status:         agent.ProposalStatusExecutionFailed,
		AutonomyTier:   agent.TierAutoExecute,
		ExecutionError: "name taken",
	})
	assert.True(t, strings.HasPrefix(failed,
		`The call to "create_report" ran on its own but it FAILED when it ran: name taken`))
	assert.NotContains(t, failed, "approved")
}
