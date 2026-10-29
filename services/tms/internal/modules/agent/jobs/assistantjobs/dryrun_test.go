package assistantjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdryrun"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDryRunStepsSetEachCallBesideTheWriteItBecame(t *testing.T) {
	t.Parallel()

	plan := &assistantservice.DryRunPlan{
		Turn: agentruntime.TurnState{Held: []string{"invoice_get", "update_invoice"}},
	}
	outcome := &agentflow.Outcome{
		Result: &agentruntimeports.RunResult{Actions: []agentruntimeports.PendingAction{
			{ToolCallID: "2", ToolName: "update_invoice", Tier: agent.TierActWithApproval},
		}},
		Events: []temporaltype.StreamItem{
			{
				Event: assistantstreamports.AssistantEventToolStarted,
				Data:  map[string]any{"callId": "1"},
			},
			{
				Event: assistantstreamports.AssistantEventToolFinished,
				Data: assistantstreamports.AssistantToolFinishedEvent{
					CallID: "1", Name: "invoice_get", Verdict: "ran",
				},
			},
			{
				Event: assistantstreamports.AssistantEventToolFinished,
				Data: &assistantstreamports.AssistantToolFinishedEvent{
					CallID: "2", Name: "update_invoice", Verdict: "proposed", Proposed: true,
				},
			},
			{Event: assistantstreamports.AssistantEventToolFinished, Data: map[string]any{
				"callId": "3", "name": "shipment_cancel", "verdict": "denied", "failed": true,
			}},
			{
				Event: assistantstreamports.AssistantEventToolFinished,
				Data: assistantstreamports.AssistantToolFinishedEvent{
					CallID: "4", Name: "shipment_get", Verdict: "ran", AgentID: pulid.MustNew("agdef_"),
				},
			},
		},
	}

	steps := dryRunSteps(plan, outcome)

	assert.Equal(t, []agentdryrun.Step{
		{CallID: "1", Tool: "invoice_get", Outcome: agentdryrun.OutcomeRuns},
		{CallID: "2", Tool: "update_invoice", Outcome: agentdryrun.OutcomeAskFirst},
		{CallID: "3", Tool: "shipment_cancel", Outcome: agentdryrun.OutcomeNotHeld},
	}, steps, "a delegate's own calls are its agent's, not the draft's")
}

func TestDryRunStepsOfNoRun(t *testing.T) {
	t.Parallel()

	assert.Empty(t, dryRunSteps(&assistantservice.DryRunPlan{}, nil))
}
