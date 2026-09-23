package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An approval that made something says what it made, and by which id.

create_report was approved and ran, and the note said only "Approved
create_report, and it ran." with the proposal's id beside it. The agent then
called describe_report with that ap_ id instead of the rd_ id of the report it
had saved, and failed. The note's first line, which the thread shows, names
the report in words; the lines after it give the agent the id to use and say
it is not the proposal's.
*/
func TestSendMessageStream_NamesWhatAnApprovedChangeMade(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "create_report",
		Status:   agent.ProposalStatusExecuted,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			Name:   "Shipments for Peak Distributing",
			IDs:    map[string]string{"definitionId": "rd_01JPEAK"},
		},
	}
	svc, conversations, _ := followUpService(t, proposal)
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	note := conversations.appended[0].Content
	first, rest, _ := strings.Cut(note, "\n")
	assert.Equal(t,
		`Approved create_report, and it ran. It created the report "Shipments for Peak Distributing".`,
		first,
	)
	assert.NotContains(t, first, "rd_01JPEAK", "the person-readable line carries no ids")
	assert.Contains(t, rest, "Decision on proposal "+proposal.ID.String()+" (create_report).")
	assert.Contains(t, rest,
		"It produced definitionId rd_01JPEAK; use that id, not the proposal's, "+
			"when you refer to what it made.")
	assert.Contains(t, rest, followUpInstruction)
}

// A result is only spoken of once the write ran. An approval still being
// carried out has made nothing yet.
func TestDecisionNote_SaysNothingMadeBeforeTheWriteRuns(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ToolName: "create_report",
		Status:   agent.ProposalStatusAccepted,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			IDs:    map[string]string{"definitionId": "rd_01"},
		},
	}

	assert.Equal(t, "Approved create_report; it is being carried out.", decisionLine(proposal))
	assert.Empty(t, producedNote(proposal))
}

// A tool that names nothing leaves the note as it was.
func TestDecisionNote_KeepsThePlainLineWithoutAResult(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ToolName: "create_dashboard",
		Status:   agent.ProposalStatusExecuted,
	}

	assert.Equal(t, "Approved create_dashboard, and it ran.", decisionLine(proposal))
	assert.Empty(t, producedNote(proposal))
}

func TestDecisionHeadline_IsTheFirstLineOnly(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Approved create_report, and it ran.",
		decisionHeadline("  Approved create_report, and it ran.\nDecision on proposal ap_1."))
	assert.Equal(t, "Rejected assign_move.", decisionHeadline("Rejected assign_move."))
	assert.Empty(t, decisionHeadline(" \n "))
}
