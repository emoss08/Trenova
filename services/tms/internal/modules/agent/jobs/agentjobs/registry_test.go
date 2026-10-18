package agentjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
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
		agentcontract.AgentRunWorkflowName,
		agentcontract.AgentScheduledRunWorkflowName,
		ReconcileSchedulesWorkflowName,
		ExpireStaleProposalsWorkflowName,
		DeleteStaleAskThreadsWorkflowName,
	} {
		assert.Contains(t, background, name)
	}
	assert.NotContains(t, background, agentcontract.AgentEvaluationWorkflowName,
		"an evaluation costs as much as a run and must not queue behind one")

	assert.Equal(t, temporaltype.TaskQueueAgentHeavy.String(), HeavyDomainConfig.TaskQueue)
	assert.Contains(t, names(workflows.heavy()), agentcontract.AgentEvaluationWorkflowName)
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
