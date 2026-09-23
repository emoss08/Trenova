package proposalrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A write that ran on its own keeps what it made on the proposal that records
// it, the way an approved one does, so a later turn reads the report's id
// rather than only that something ran.
func TestRecord_KeepsWhatAnAutomaticWriteMade(t *testing.T) {
	t.Parallel()

	executed := pendingAction("create_report")
	executed.Tier = agent.TierAutoExecute
	executed.Executed = true
	executed.ExecutionResult = &agent.ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   "Lane revenue",
		IDs:    map[string]string{"definitionId": "rd_01"},
	}

	failed := pendingAction("create_report")
	failed.Tier = agent.TierAutoExecute
	failed.Executed = true
	failed.ExecutionError = "name taken"
	failed.ExecutionResult = &agent.ToolExecutionResult{Action: "created", Kind: "report"}

	result := recordActions(t, &capturingPlans{}, executed, failed)

	require.Len(t, result.Proposals, 2)
	made := result.Proposals[0]
	assert.Equal(t, agent.ProposalStatusExecuted, made.Status)
	require.NotNil(t, made.ExecutionResult)
	assert.Equal(t, "Lane revenue", made.ExecutionResult.Name)
	assert.Equal(t, map[string]string{"definitionId": "rd_01"}, made.ExecutionResult.IDs)

	assert.Equal(t, agent.ProposalStatusExecutionFailed, result.Proposals[1].Status)
	assert.Nil(t, result.Proposals[1].ExecutionResult, "a failed write made nothing")
}
