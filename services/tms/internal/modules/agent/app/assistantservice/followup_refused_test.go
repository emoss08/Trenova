package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/decisionnotes"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessageStream_SaysAnApprovalWasRefusedBecauseItsRecordMoved(t *testing.T) {
	t.Parallel()

	proposal := assistanttest.RefusedProposal()
	svc, conversations, completion := followUpService(t, proposal)
	anchors := &assistanttest.RecordedAnchors{}
	svc.anchors = anchors
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(t,
		"Approved shipment_place_hold, but it was not carried out: the record changed since "+
			"this was proposed: the shipment SEED-BENCH-CM-01 was at version 4 when this was "+
			"proposed and is at version 5 now, so nothing was changed",
		decisionnotes.DecisionHeadline(conversations.Appended[0].Content),
		"the person reads that nothing happened, and why",
	)

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1].Content
	assert.Contains(t, last, "Do not retry the refused proposal")
	assert.Contains(t, last, "offer to propose it again against its current version")
	assert.NotContains(t, last, decisionnotes.FollowUpInstruction)

	require.NotEmpty(t, anchors.Asked)
	refused := anchors.Asked[len(anchors.Asked)-1]
	require.Len(t, refused.Records, 1)
	assert.Equal(t, proposal.TargetID.String(), refused.Records[0].ID)
	assert.Equal(t, actor.UserID, refused.Actor.UserID, "the record is read as the person")
	assert.Contains(t, last, "Range pulled the load by phone",
		"the agent reads the record as it is now beside the note")
}

func TestSendMessageStream_SaysNothingInAPlanRanWhenARecordMoved(t *testing.T) {
	t.Parallel()

	failedStep := 2
	plan := &agentplan.AgentPlan{
		ID:           pulid.MustNew("apl_"),
		Title:        "Dock 4 notes for tomorrow",
		Status:       agentplan.PlanStatusFailed,
		StepCount:    3,
		FailedStep:   &failedStep,
		FailureError: "nothing ran, because a record the plan changes moved on after it was proposed",
	}
	moved := assistanttest.RefusedProposal()
	moved.ToolName = "shipment_add_comment"
	moved.PlanID = &plan.ID
	moved.PlanStep = 2
	steps := []*agent.AgentProposal{
		{ID: pulid.MustNew("ap_"), ToolName: "shipment_add_comment", PlanID: &plan.ID,
			PlanStep: 1, Status: agent.ProposalStatusSkipped},
		moved,
		{ID: pulid.MustNew("ap_"), ToolName: "shipment_add_comment", PlanID: &plan.ID,
			PlanStep: 3, Status: agent.ProposalStatusSkipped},
	}
	svc, conversations, completion := followUpService(t, moved)
	svc.proposals = &assistanttest.StubProposalRepo{ByThread: steps}
	svc.plans = &stubPlanStore{plans: []*agentplan.AgentPlan{plan}}
	anchors := &assistanttest.RecordedAnchors{}
	svc.anchors = anchors
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.Thread.ID,
		FollowUpPlanID: plan.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	headline := decisionnotes.DecisionHeadline(conversations.Appended[0].Content)
	assert.True(t, strings.HasPrefix(headline,
		`Approved the plan "Dock 4 notes for tomorrow", but nothing ran: a record it changes `+
			`moved on after it was proposed. Step 2 (shipment_add_comment): the record changed`),
		headline)
	assert.NotContains(t, headline, "steps after it were skipped",
		"a plan refused before it ran did not run the steps before the refused one either")

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1].Content
	assert.Contains(t, last, "Do not retry the refused proposal")
	require.NotEmpty(t, anchors.Asked)
	assert.Equal(t, moved.TargetID.String(), anchors.Asked[len(anchors.Asked)-1].Records[0].ID)
}
