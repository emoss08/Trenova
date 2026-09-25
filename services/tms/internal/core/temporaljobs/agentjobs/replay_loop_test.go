package agentjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/temporaljobs/replaytest"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	loopReplayRead  = "get_shipment"
	loopReplayWrite = "assign_move"
)

func loopReplayRuntime() *agentruntime.Service {
	return agentruntime.New(agentruntime.Params{
		Logger:     zap.NewNop(),
		Completion: &agentruntimetest.ScriptedCompletion{},
		QueryTools: &agentruntimetest.StubQueryRegistry{
			Tools: []serviceports.AgentQueryTool{&agentruntimetest.StubQueryTool{
				ToolName: loopReplayRead,
			}},
		},
		ActionTools: &agentruntimetest.StubActionRegistry{
			Tools: []serviceports.AgentTool{&agentruntimetest.StubActionTool{
				ToolName: loopReplayWrite,
			}},
		},
		Permissions: &agentruntimetest.StubPermissions{},
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
			workflow.RegisterOptions{Name: AgentRunWorkflowName},
		)
	})
}
