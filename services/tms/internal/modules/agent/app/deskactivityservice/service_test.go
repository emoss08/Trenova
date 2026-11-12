package deskactivityservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskactivityports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(v int64) *int64 { return &v }

func proposal(id, run, tool string, tier agent.AutonomyTier, executedAt int64) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.ID(id),
		RunID:          pulid.ID(run),
		ToolName:       tool,
		AutonomyTier:   tier,
		Status:         agent.ProposalStatusExecuted,
		ExecutedAt:     at(executedAt),
		TargetResource: "shipment",
		TargetID:       pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ" + id[len(id)-1:]),
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "filed",
			Kind:   "rate confirmation",
		},
	}
}

func TestFold_FoldsARunOfTheSameWriteIntoOneLine(t *testing.T) {
	entries := fold([]*agent.AgentProposal{
		proposal("ap_3", "arun_1", "triage_file_document", agent.TierAutoExecute, 30),
		proposal("ap_2", "arun_1", "triage_file_document", agent.TierAutoExecute, 20),
		proposal("ap_1", "arun_2", "triage_file_document", agent.TierActWithApproval, 10),
	})

	require.Len(t, entries, 2)
	assert.Equal(t, deskactivityports.OutcomeOnItsOwn, entries[0].Outcome)
	assert.Equal(t, 2, entries[0].Count)
	assert.Equal(t, int64(30), entries[0].At, "a line is as new as its newest write")
	assert.Equal(t, "Filed 2 rate confirmations", entries[0].Title)
	assert.Len(t, entries[0].Records, 2)
	assert.Equal(t, deskactivityports.OutcomeApproved, entries[1].Outcome)
}

func TestOutcomeOf_NeverClaimsMoreThanTheRecordSays(t *testing.T) {
	failed := proposal("ap_1", "arun_1", "invoice_post", agent.TierActWithApproval, 10)
	failed.ExecutionError = "The period is closed."
	previewed := proposal("ap_2", "arun_1", "invoice_post", agent.TierAutoExecute, 0)
	previewed.ExecutedAt = nil
	previewed.SimulatedAt = at(5)

	assert.Equal(t, deskactivityports.OutcomeFailed, outcomeOf(failed))
	assert.Equal(t, deskactivityports.OutcomePreviewed, outcomeOf(previewed))
	assert.Equal(t, "The period is closed.", detailOf(failed))
}

func TestFold_APlansStepsAreOneLine(t *testing.T) {
	planID := pulid.ID("apl_01JABCDEFGHJKMNPQRSTVWXYZ0")
	first := proposal("ap_1", "arun_1", "stop_update_appointment", agent.TierActWithApproval, 20)
	second := proposal("ap_2", "arun_1", "dispatch_assign_worker", agent.TierActWithApproval, 21)
	first.PlanID, second.PlanID = &planID, &planID

	entries := fold([]*agent.AgentProposal{second, first})

	require.Len(t, entries, 1)
	assert.Equal(t, planID, entries[0].PlanID)
	assert.Equal(t, 2, entries[0].Count)
}

func TestAddRecord_CountsPastTheFirstFive(t *testing.T) {
	entry := &deskactivityports.Entry{}
	for idx := range recordsPerEntry + 2 {
		p := proposal("ap_1", "arun_1", "triage_file_document", agent.TierAutoExecute, 1)
		p.TargetID = pulid.MustNew("shp_")
		_ = idx
		addRecord(entry, p)
	}

	assert.Len(t, entry.Records, recordsPerEntry)
	assert.Equal(t, 2, entry.RecordsMore)
}

func TestWhatOf_NamesWhatTheWriteMadeOrFallsBackToTheTool(t *testing.T) {
	made := proposal("ap_1", "arun_1", "triage_file_document", agent.TierAutoExecute, 10)
	made.ExecutionResult.Name = "RC-1042"
	assert.Equal(t, "Filed rate confirmation RC-1042", whatOf(made))

	unnamed := proposal("ap_2", "arun_1", "triage_file_document", agent.TierAutoExecute, 10)
	assert.Equal(t, "Filed rate confirmation", whatOf(unnamed))

	silent := proposal("ap_3", "arun_1", "triage_file_document", agent.TierAutoExecute, 10)
	silent.ExecutionResult = nil
	fallback := whatOf(silent)
	assert.NotEmpty(t, fallback)
	assert.NotContains(t, fallback, "_")
}
