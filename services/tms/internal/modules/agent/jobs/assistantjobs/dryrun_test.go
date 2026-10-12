package assistantjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdryrun"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
		Result: &agentports.RunResult{Actions: []agentports.PendingAction{
			{ToolCallID: "2", ToolName: "update_invoice", Tier: agent.TierActWithApproval},
		}},
		Events: []temporaltype.StreamItem{
			{Event: agentports.AssistantEventToolStarted, Data: map[string]any{"callId": "1"}},
			{Event: agentports.AssistantEventToolFinished, Data: agentports.AssistantToolFinishedEvent{
				CallID: "1", Name: "get_invoice", Verdict: "ran",
			}},
			{Event: agentports.AssistantEventToolFinished, Data: &agentports.AssistantToolFinishedEvent{
				CallID: "2", Name: "update_invoice", Verdict: "proposed", Proposed: true,
			}},
			{Event: agentports.AssistantEventToolFinished, Data: map[string]any{
				"callId": "3", "name": "cancel_shipment", "verdict": "denied", "failed": true,
			}},
			{Event: agentports.AssistantEventToolFinished, Data: agentports.AssistantToolFinishedEvent{
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
