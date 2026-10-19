package assistantjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/replaytest"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const replayToolName = "shipment_get"

func replayRuntime() *agentruntime.Service {
	return agentruntime.New(agentruntime.Params{
		Logger:     zap.NewNop(),
		Completion: &agenttest.ScriptedCompletion{},
		QueryTools: &agenttest.StubQueryRegistry{
			Tools: []agentports.AgentQueryTool{&agenttest.StubQueryTool{
				ToolName: replayToolName,
			}},
		},
		ActionTools: &agenttest.StubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
	})
}

func replayPayload() *assistantcontract.AssistantTurnPayload {
	org := pulid.MustNew("org_")
	bu := pulid.MustNew("bu_")
	user := pulid.MustNew("usr_")

	return &assistantcontract.AssistantTurnPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: org,
			BusinessUnitID: bu,
			UserID:         user,
		},
		TurnID:   pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
		Content:  "Where is load 12345?",
		Actor: permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    user,
			UserID:         user,
			OrganizationID: org,
			BusinessUnitID: bu,
		},
	}
}

func replayPlan(
	t *testing.T,
	runtime *agentruntime.Service,
	payload *assistantcontract.AssistantTurnPayload,
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

func TestAssistantTurnWorkflow_ReplaysRecordedHistories(t *testing.T) {
	t.Parallel()

	replaytest.Dir(t, filepath.Join("testdata", "replay"), func(r worker.WorkflowReplayer) {
		r.RegisterWorkflowWithOptions(
			NewWorkflows(replayRuntime()).AssistantTurnWorkflow,
			workflow.RegisterOptions{Name: assistantcontract.AssistantTurnWorkflowName},
		)
	})
}
