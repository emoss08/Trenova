package agentjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/replaytest"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	loopReplayRead  = "shipment_get"
	loopReplayWrite = "dispatch_assign_move"
)

func loopReplayRuntime() *agentruntime.Service {
	return agentruntime.New(agentruntime.Params{
		Logger:     zap.NewNop(),
		Completion: &agenttest.ScriptedCompletion{},
		QueryTools: &agenttest.StubQueryRegistry{
			Tools: []agentports.AgentQueryTool{&agenttest.StubQueryTool{
				ToolName: loopReplayRead,
			}},
		},
		ActionTools: &agenttest.StubActionRegistry{
			Tools: []agentports.AgentTool{&agenttest.StubActionTool{
				ToolName: loopReplayWrite,
			}},
		},
		Permissions: &agenttest.StubPermissions{},
	})
}

func loopReplayDefinition(id pulid.ID) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              id,
		Name:            "Recorded desk",
		Instructions:    "Keep moves covered.",
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       []string{loopReplayRead, loopReplayWrite},
		TriggerMode:     agentdefinition.TriggerEvent,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	return definition
}

func TestAgentRunWorkflow_ReplaysRecordedLoopHistories(t *testing.T) {
	t.Parallel()

	replaytest.Dir(t, filepath.Join("testdata", "replay-loop"), func(r worker.WorkflowReplayer) {
		r.RegisterWorkflowWithOptions(
			NewWorkflows(loopReplayRuntime()).AgentRunWorkflow,
			workflow.RegisterOptions{Name: agentcontract.AgentRunWorkflowName},
		)
	})
}
