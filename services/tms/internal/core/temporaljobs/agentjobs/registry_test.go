package agentjobs

import (
	"testing"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func queueByName(definitions []temporaltype.WorkflowDefinition) map[string]string {
	queues := make(map[string]string, len(definitions))
	for _, definition := range definitions {
		queues[definition.Name] = definition.TaskQueue
	}

	return queues
}

func TestRegisterWorkflows_KeepsTheWorkloadClassesApart(t *testing.T) {
	t.Parallel()

	background := queueByName(RegisterBackgroundWorkflows())
	for _, name := range []string{
		AgentRunWorkflowName,
		AgentSweepWorkflowName,
		ExpireStaleProposalsWorkflowName,
		DeleteStaleAskThreadsWorkflowName,
	} {
		assert.Equal(t, temporaltype.TaskQueueAgentBackground.String(), background[name], name)
	}
	assert.NotContains(t, background, AgentEvaluationWorkflowName,
		"an evaluation costs as much as a run and must not queue behind one")

	heavy := queueByName(RegisterHeavyWorkflows())
	assert.Equal(t,
		temporaltype.TaskQueueAgentHeavy.String(),
		heavy[AgentEvaluationWorkflowName],
	)
}

func TestRegisterDrainWorkflows_CoversEverythingOnTheOldQueue(t *testing.T) {
	t.Parallel()

	// A workflow's task queue is fixed when it starts, so runs already in
	// flight when the split deployed still dispatch to agent-queue. Every
	// workflow has to stay reachable there until they have drained, or they
	// hang until their timeouts fire.
	drain := queueByName(RegisterDrainWorkflows())
	require.Len(t, drain, len(RegisterBackgroundWorkflows())+len(RegisterHeavyWorkflows()))

	for name, queue := range drain {
		assert.Equal(t, temporaltype.TaskQueueAgent.String(), queue, name)
	}
	assert.Contains(t, drain, AgentRunWorkflowName)
	assert.Contains(t, drain, AgentEvaluationWorkflowName)
}

func TestSchedules_RunOnTheBackgroundQueue(t *testing.T) {
	t.Parallel()

	// schedule.Schedule.Hash covers the task queue and the reconciler replaces
	// the whole action, so changing this constant re-points the existing
	// schedules on the next boot rather than needing new ids.
	for _, scheduled := range NewScheduleProvider().GetSchedules() {
		assert.Equal(t,
			temporaltype.TaskQueueAgentBackground.String(),
			scheduled.TaskQueue,
			scheduled.ID,
		)
	}
}
