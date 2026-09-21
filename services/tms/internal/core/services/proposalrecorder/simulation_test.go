package proposalrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A simulated automatic write is recorded as what it would have done, with
// nothing left for a person to decide and no plan formed around it.
func TestRecord_KeepsASimulatedWriteAsSimulated(t *testing.T) {
	t.Parallel()

	simulated := pendingAction("update_tractor_status")
	simulated.Tier = agent.TierAutoExecute
	simulated.Simulated = true
	simulated.Simulation = &agent.ToolSimulation{
		Summary:   "Would set 1 tractor to OutOfService.",
		Changes:   []agent.FieldChange{{Field: "tractor 101", From: "Available", To: "OutOfService"}},
		Previewed: true,
	}

	plans := &capturingPlans{}
	result := recordActions(t, plans, simulated, pendingAction("assign_move"))

	require.Len(t, result.Proposals, 2)
	first := result.Proposals[0]
	assert.Equal(t, agent.ProposalStatusSimulated, first.Status)
	require.NotNil(t, first.SimulatedAt)
	require.NotNil(t, first.Simulation)
	assert.Equal(t, "Would set 1 tractor to OutOfService.", first.Simulation.Summary)
	assert.Nil(t, first.ExecutedAt, "a simulation is not an execution")

	assert.Nil(t, result.Plan, "one pending write beside a simulated one is no plan")
	assert.Equal(t, agent.ProposalStatusPending, result.Proposals[1].Status)
	assert.Nil(t, result.Proposals[1].PlanID)
}
