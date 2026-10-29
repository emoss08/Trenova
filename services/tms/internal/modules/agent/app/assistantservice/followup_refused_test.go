package assistantservice

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedAnchors struct {
	asked []*agentports.RecordAnchorRequest
}

func (r *recordedAnchors) Anchor(
	_ context.Context,
	req *agentports.RecordAnchorRequest,
) []agentturn.RuntimeAnchor {
	r.asked = append(r.asked, req)
	out := make([]agentturn.RuntimeAnchor, 0, len(req.Records))
	for _, record := range req.Records {
		out = append(out, agentturn.RuntimeAnchor{
			Kind:  "shipment",
			ID:    record.ID,
			Label: "SEED-BENCH-CM-01",
			Facts: []agentturn.AnchorFact{
				{Name: "status", Value: "Canceled for Range Logistics"},
				{Name: "canceled", Value: "by Admin: Range pulled the load by phone"},
			},
		})
	}

	return out
}

func refusedProposal() *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		ToolName:       "shipment_place_hold",
		Status:         agent.ProposalStatusExecutionFailed,
		TargetResource: "shipment",
		TargetID:       pulid.MustNew("shp_"),
		TargetVersion:  4,
		ExecutionError: fmt.Errorf(
			"%w: the shipment SEED-BENCH-CM-01 was at version 4 when this was proposed and is "+
				"at version 5 now, so nothing was changed",
			agentplan.ErrTargetChanged,
		).Error(),
	}
}

func TestSendMessageStream_SaysAnApprovalWasRefusedBecauseItsRecordMoved(t *testing.T) {
	t.Parallel()

	proposal := refusedProposal()
	svc, conversations, completion := followUpService(t, proposal)
	anchors := &recordedAnchors{}
	svc.anchors = anchors
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(t,
		"Approved shipment_place_hold, but it was not carried out: the record changed since "+
			"this was proposed: the shipment SEED-BENCH-CM-01 was at version 4 when this was "+
			"proposed and is at version 5 now, so nothing was changed",
		decisionHeadline(conversations.appended[0].Content),
		"the person reads that nothing happened, and why",
	)

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1].Content
	assert.Contains(t, last, "Do not retry the refused proposal")
	assert.Contains(t, last, "offer to propose it again against its current version")
	assert.NotContains(t, last, followUpInstruction)

	require.NotEmpty(t, anchors.asked)
	refused := anchors.asked[len(anchors.asked)-1]
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
	moved := refusedProposal()
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
	svc.proposals = &stubProposalRepo{byThread: steps}
	svc.plans = &stubPlanStore{plans: []*agentplan.AgentPlan{plan}}
	anchors := &recordedAnchors{}
	svc.anchors = anchors
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.thread.ID,
		FollowUpPlanID: plan.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	headline := decisionHeadline(conversations.appended[0].Content)
	assert.True(t, strings.HasPrefix(headline,
		`Approved the plan "Dock 4 notes for tomorrow", but nothing ran: a record it changes `+
			`moved on after it was proposed. Step 2 (shipment_add_comment): the record changed`),
		headline)
	assert.NotContains(t, headline, "steps after it were skipped",
		"a plan refused before it ran did not run the steps before the refused one either")

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1].Content
	assert.Contains(t, last, "Do not retry the refused proposal")
	require.NotEmpty(t, anchors.asked)
	assert.Equal(t, moved.TargetID.String(), anchors.asked[len(anchors.asked)-1].Records[0].ID)
}

func TestAnchorChanged_ReadsOnlyTheRefusedRecordsNotAlreadyInPlay(t *testing.T) {
	t.Parallel()

	inPlay := pulid.MustNew("shp_").String()
	unread := pulid.MustNew("shp_").String()
	anchors := &recordedAnchors{}
	svc := &Service{anchors: anchors}
	actor := testActor()

	got := svc.anchorChanged(t.Context(),
		[]agentturn.RuntimeAnchor{{Kind: "shipment", ID: inPlay}},
		[]agentpagecontext.EntityRef{{ID: inPlay}, {ID: unread}, {ID: unread}},
		actor, actor.TenantInfo())

	require.Len(t, anchors.asked, 1)
	require.Len(t, anchors.asked[0].Records, 1, "a record already in play is not read twice")
	assert.Equal(t, unread, anchors.asked[0].Records[0].ID)
	require.Len(t, got, 2)
	assert.Equal(t, unread, got[1].ID)

	untouched := svc.anchorChanged(t.Context(), got, nil, actor, actor.TenantInfo())
	assert.Equal(t, got, untouched)
	assert.Len(t, anchors.asked, 1, "nothing refused, nothing read")
}

func TestRefusedOverChange_IsOnlyAnApprovalRefusedOverAMovedRecord(t *testing.T) {
	t.Parallel()

	moved := refusedProposal()
	failed := &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusExecutionFailed,
		ExecutionError: "validation failed: bol: taken",
	}
	executed := &agent.AgentProposal{ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusExecuted}

	refused := refusedOverChange([]*agent.AgentProposal{failed, moved, executed, nil})

	require.Len(t, refused, 1)
	assert.Equal(t, moved.ID, refused[0].ID)
}
