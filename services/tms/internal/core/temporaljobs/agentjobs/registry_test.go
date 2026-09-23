package agentjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func names(definitions []registry.WorkflowDefinition) map[string]struct{} {
	found := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		found[definition.Name] = struct{}{}
	}

	return found
}

func TestWorkflows_KeepTheWorkloadClassesApart(t *testing.T) {
	t.Parallel()

	workflows := NewWorkflows(nil)

	assert.Equal(
		t,
		temporaltype.TaskQueueAgentBackground.String(),
		BackgroundDomainConfig.TaskQueue,
	)
	background := names(workflows.background())
	for _, name := range []string{
		AgentRunWorkflowName,
		AgentScheduledRunWorkflowName,
		ReconcileSchedulesWorkflowName,
		ExpireStaleProposalsWorkflowName,
		DeleteStaleAskThreadsWorkflowName,
	} {
		assert.Contains(t, background, name)
	}
	assert.NotContains(t, background, AgentEvaluationWorkflowName,
		"an evaluation costs as much as a run and must not queue behind one")

	assert.Equal(t, temporaltype.TaskQueueAgentHeavy.String(), HeavyDomainConfig.TaskQueue)
	assert.Contains(t, names(workflows.heavy()), AgentEvaluationWorkflowName)
}

func TestWorkflows_DrainCoversEverythingOnTheOldQueue(t *testing.T) {
	t.Parallel()

	// A workflow's task queue is fixed when it starts, so runs already in
	// flight when the split deployed still dispatch to agent-queue. Every
	// workflow has to stay reachable there until they have drained, or they
	// hang until their timeouts fire.
	workflows := NewWorkflows(nil)
	drain := names(workflows.drain())
	require.Len(t, drain, len(workflows.background())+len(workflows.heavy()))
	assert.Equal(t, temporaltype.TaskQueueAgent.String(), DrainDomainConfig.TaskQueue)
	assert.Contains(t, drain, AgentRunWorkflowName)
	assert.Contains(t, drain, AgentEvaluationWorkflowName)
}

// A workflow a schedule starts is started by its function, so the name it is
// registered under must be the one the SDK derives from that function.
func TestSchedules_StartWorkflowsUnderTheNamesTheyAreRegisteredBy(t *testing.T) {
	t.Parallel()

	registered := names(NewWorkflows(nil).background())
	for _, scheduled := range NewScheduleProvider().GetSchedules() {
		assert.Equal(t,
			temporaltype.TaskQueueAgentBackground.String(),
			scheduled.TaskQueue,
			scheduled.ID,
		)
		name := scheduled.GetWorkflowName()
		short := name[len(name)-len(lastSegment(name)):]
		assert.Contains(t, registered, short, scheduled.ID)
	}
}

func lastSegment(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}

	return name
}
