package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/decisionnotes"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An approval that made something says what it made, and by which id.

report_create was approved and ran, and the note said only "Approved
report_create, and it ran." with the proposal's id beside it. The agent then
called report_get with that ap_ id instead of the rd_ id of the report it
had saved, and failed. The note's first line, which the thread shows, names
the report in words; the lines after it give the agent the id to use and say
it is not the proposal's.
*/
func TestSendMessageStream_NamesWhatAnApprovedChangeMade(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "report_create",
		Status:   agent.ProposalStatusExecuted,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			Name:   "Shipments for Peak Distributing",
			IDs:    map[string]string{"definitionId": "rd_01JPEAK"},
		},
	}
	svc, conversations, _ := followUpService(t, proposal)
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	note := conversations.Appended[0].Content
	first, rest, _ := strings.Cut(note, "\n")
	assert.Equal(
		t,
		`Approved report_create, and it ran. It created the report "Shipments for Peak Distributing".`,
		first,
	)
	assert.NotContains(t, first, "rd_01JPEAK", "the person-readable line carries no ids")
	assert.Contains(t, rest, "Decision on proposal "+proposal.ID.String()+" (report_create).")
	assert.Contains(t, rest,
		"It produced definitionId rd_01JPEAK; use that id, not the proposal's, "+
			"when you refer to what it made.")
	assert.Contains(t, rest, decisionnotes.FollowUpInstruction)
}

/*
"Void and recreate" ran as one plan, and the follow-up said only that both
steps ran, so the agent learned the new pro number by searching for it. The
note now says what each step that ran made, in words on the line the thread
shows and by id on the agent's lines, step by step.
*/
func TestSendMessageStream_NamesWhatEachStepOfAPlanMade(t *testing.T) {
	t.Parallel()

	plan := &agentplan.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		Title:          "Dispatch: 2 changes",
		Status:         agentplan.PlanStatusCompleted,
		StepCount:      2,
		CompletedSteps: 2,
	}
	planID := plan.ID
	voided := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "void_shipment",
		Status:   agent.ProposalStatusExecuted,
		PlanID:   &planID,
		PlanStep: 1,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "voided", Kind: "shipment", Name: "PRO-1001",
			IDs: map[string]string{"shipmentId": "shp_old"},
		},
	}
	copied := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "shipment_duplicate",
		Status:   agent.ProposalStatusExecuted,
		PlanID:   &planID,
		PlanStep: 2,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "created", Kind: "shipment", Name: "PRO-1002",
			IDs: map[string]string{"shipmentId": "shp_new"},
		},
	}
	svc, conversations, _ := followUpService(t, copied)
	svc.proposals = &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
		copied, voided, {ID: pulid.MustNew("ap_"), ToolName: "dispatch_assign_move"},
	}}
	svc.plans = &stubPlanStore{plans: []*agentplan.AgentPlan{plan}}
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.Thread.ID,
		FollowUpPlanID: plan.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	first, rest, _ := strings.Cut(conversations.Appended[0].Content, "\n")
	assert.Equal(t,
		`Approved the plan "Dispatch: 2 changes", and all 2 steps ran. `+
			`Step 1: It voided the shipment "PRO-1001". `+
			`Step 2: It created the shipment "PRO-1002".`,
		first,
	)
	assert.NotContains(t, first, "shp_new")
	assert.Contains(t, rest, "Decision on plan "+plan.ID.String()+" (2 steps).")
	assert.Contains(t, rest, "Step 2 (shipment_duplicate): It produced shipmentId shp_new;")
	assert.Less(t, strings.Index(rest, "Step 1 (void_shipment)"),
		strings.Index(rest, "Step 2 (shipment_duplicate)"), "steps are told in order")
	assert.Contains(t, rest, decisionnotes.FollowUpInstruction)
}
