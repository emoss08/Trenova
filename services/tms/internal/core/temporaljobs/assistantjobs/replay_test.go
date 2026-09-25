package assistantjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/replaytest"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const replayToolName = "get_shipment"

// replayRuntime is the runtime the recorded turns were driven by: one read
// tool, held by the agent, so the loop dispatches it as an activity.
func replayRuntime() *agentruntime.Service {
	return agentruntime.New(agentruntime.Params{
		Logger:     zap.NewNop(),
		Completion: &agentruntimetest.ScriptedCompletion{},
		QueryTools: &agentruntimetest.StubQueryRegistry{
			Tools: []serviceports.AgentQueryTool{&agentruntimetest.StubQueryTool{
				ToolName: replayToolName,
			}},
		},
		ActionTools: &agentruntimetest.StubActionRegistry{},
		Permissions: &agentruntimetest.StubPermissions{},
	})
}

func replayPlan(
	t *testing.T,
	runtime *agentruntime.Service,
	payload *AssistantTurnPayload,
) *assistantservice.TurnPlan {
	t.Helper()

	definition := &agentdefinition.Definition{
		Name:            "Recorded desk",
		Instructions:    "Help dispatch.",
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       []string{replayToolName},
		Enabled:         true,
	}
	definition.ApplyDefaults()

	plan := &assistantservice.TurnPlan{
		ThreadID:   payload.ThreadID,
		Definition: definition,
		Input:      payload.Content,
		Decision:   agentguard.Decision{Allowed: true},
	}
	plan.Turn = runtime.OpenTurn(t.Context(), plan.RunRequest(&payload.Actor)).State()

	return plan
}

// Turns like these were in flight when the tracing and provenance fields
// shipped: recorded by replay_record_test.go from the code before it, on a real
// server. Every change since added data to activity inputs, stream items and
// results, and none may add, remove or reorder a command, so each must still
// replay.
func TestAssistantTurnWorkflow_ReplaysRecordedHistories(t *testing.T) {
	t.Parallel()

	replaytest.Dir(t, filepath.Join("testdata", "replay"), func(r worker.WorkflowReplayer) {
		r.RegisterWorkflowWithOptions(
			NewWorkflows(replayRuntime()).AssistantTurnWorkflow,
			workflow.RegisterOptions{Name: AssistantTurnWorkflowName},
		)
	})
}
