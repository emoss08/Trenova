package assistantjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdryrun"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDryRunStepsSetEachCallBesideTheWriteItBecame(t *testing.T) {
	t.Parallel()

	plan := &assistantservice.DryRunPlan{
		Turn: agentruntime.TurnState{Held: []string{"get_invoice", "update_invoice"}},
	}
	outcome := &agentflow.Outcome{
		Result: &serviceports.RunResult{Actions: []serviceports.PendingAction{
			{ToolCallID: "2", ToolName: "update_invoice", Tier: agent.TierActWithApproval},
		}},
		Events: []temporaltype.StreamItem{
			{Event: serviceports.AssistantEventToolStarted, Data: map[string]any{"callId": "1"}},
			{Event: serviceports.AssistantEventToolFinished, Data: serviceports.AssistantToolFinishedEvent{
				CallID: "1", Name: "get_invoice", Verdict: "ran",
			}},
			{Event: serviceports.AssistantEventToolFinished, Data: &serviceports.AssistantToolFinishedEvent{
				CallID: "2", Name: "update_invoice", Verdict: "proposed", Proposed: true,
			}},
			{Event: serviceports.AssistantEventToolFinished, Data: map[string]any{
				"callId": "3", "name": "cancel_shipment", "verdict": "denied", "failed": true,
			}},
			{Event: serviceports.AssistantEventToolFinished, Data: serviceports.AssistantToolFinishedEvent{
				CallID: "4", Name: "get_shipment", Verdict: "ran", AgentID: pulid.MustNew("agdef_"),
			}},
		},
	}

	steps := dryRunSteps(plan, outcome)

	assert.Equal(t, []agentdryrun.Step{
		{CallID: "1", Tool: "get_invoice", Outcome: agentdryrun.OutcomeRuns},
		{CallID: "2", Tool: "update_invoice", Outcome: agentdryrun.OutcomeAskFirst},
		{CallID: "3", Tool: "cancel_shipment", Outcome: agentdryrun.OutcomeNotHeld},
	}, steps, "a delegate's own calls are its agent's, not the draft's")
}

func TestDryRunStepsOfNoRun(t *testing.T) {
	t.Parallel()

	assert.Empty(t, dryRunSteps(&assistantservice.DryRunPlan{}, nil))
}
