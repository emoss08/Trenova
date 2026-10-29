package assistantproposals

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedUsers struct {
	tenantports.UserRepository
	users []*tenant.User
	err   error
	asked [][]pulid.ID
}

func (u *namedUsers) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	u.asked = append(u.asked, req.UserIDs)
	if u.err != nil {
		return nil, u.err
	}

	return u.users, nil
}

type decidedPlanFixture struct {
	svc       *Service
	thread    *conversation.Thread
	decided   *agentplan.AgentPlan
	waiting   *agentplan.AgentPlan
	deciderID pulid.ID
}

func newDecidedPlanFixture(users *namedUsers) decidedPlanFixture {
	run := &agent.AgentRun{ID: pulid.MustNew("arun_")}
	runs := &assistanttest.StubRunRepo{ByID: map[pulid.ID]*agent.AgentRun{run.ID: run}}
	deciderID := pulid.MustNew("usr_")
	decidedAt, commitsAt, undoneAt := int64(
		1_760_000_060,
	), int64(
		1_760_000_070,
	), int64(
		1_760_000_065,
	)
	failed := 3
	decided := &agentplan.AgentPlan{
		ID:                    pulid.MustNew("apl_"),
		RunID:                 run.ID,
		Title:                 "Recover S-240201 after the Tyson dock delay",
		Status:                agentplan.PlanStatusFailed,
		StepCount:             4,
		CompletedSteps:        2,
		FailedStep:            &failed,
		FailureError:          "a posted invoice cannot take new charges",
		DecidedByUserID:       &deciderID,
		DecidedAt:             &decidedAt,
		CommitsAt:             &commitsAt,
		UndoneAt:              &undoneAt,
		ApprovalNote:          "Go ahead, Samuel is already at the yard.",
		ApprovalPreviewDigest: "7f3a9c21e",
	}
	waiting := &agentplan.AgentPlan{
		ID:        pulid.MustNew("apl_"),
		RunID:     run.ID,
		Title:     "Dispatch: 2 changes",
		Status:    agentplan.PlanStatusPending,
		StepCount: 2,
	}
	decidedID, waitingID := decided.ID, waiting.ID
	shared := pulid.MustNew("shp_")
	proposals := &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
		{ID: pulid.MustNew("ap_"), RunID: run.ID, PlanID: &decidedID, PlanStep: 2,
			TargetResource: "shipment", TargetID: shared},
		{ID: pulid.MustNew("ap_"), RunID: run.ID, PlanID: &decidedID, PlanStep: 1,
			TargetResource: "shipment", TargetID: shared},
		{ID: pulid.MustNew("ap_"), RunID: run.ID, PlanID: &waitingID, PlanStep: 1,
			TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
		{ID: pulid.MustNew("ap_"), RunID: run.ID, PlanID: &waitingID, PlanStep: 2,
			TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
		{ID: pulid.MustNew("ap_"), RunID: run.ID},
	}}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	svc := newProposalReadsBehind(runs, proposals,
		&assistanttest.StubConversationRepo{Thread: thread}, assistanttest.ShadowSwitches{})
	svc.plans = &threadPlans{plans: []*agentplan.AgentPlan{decided, waiting}}
	if users != nil {
		svc.users = users
	}

	return decidedPlanFixture{
		svc:       svc,
		thread:    thread,
		decided:   decided,
		waiting:   waiting,
		deciderID: deciderID,
	}
}

func TestListThreadPlans_SaysHowEachPlanRunsAndWhoDecidedItWithWhat(t *testing.T) {
	t.Parallel()

	users := &namedUsers{}
	fx := newDecidedPlanFixture(users)
	users.users = []*tenant.User{{ID: fx.deciderID, Name: "Elena Ruiz"}}

	listed, err := fx.svc.ListThreadPlans(
		t.Context(),
		conversationports.GetThreadRequest{ID: fx.thread.ID},
	)
	require.NoError(t, err)
	require.Len(t, listed, 2)

	decided := listed[0]
	assert.Equal(t, fx.decided.ID, decided.ID)
	assert.Equal(t, agentplan.PlanRunModeInOrder, decided.RunMode,
		"two steps change one shipment, so the plan stops at the first failure")
	assert.Equal(t, fx.deciderID, decided.DecidedByUserID)
	assert.Equal(t, "Elena Ruiz", decided.DecidedByName)
	assert.Equal(t, "Go ahead, Samuel is already at the yard.", decided.DecisionNote)
	assert.Equal(t, "7f3a9c21e", decided.PreviewDigest)
	require.NotNil(t, decided.CommitsAt)
	assert.Equal(t, int64(1_760_000_070), *decided.CommitsAt)
	require.NotNil(t, decided.UndoneAt)
	assert.Equal(t, int64(1_760_000_065), *decided.UndoneAt)

	waiting := listed[1]
	assert.Equal(t, agentplan.PlanRunModeEachStep, waiting.RunMode,
		"each step changes a different invoice, so every step runs")
	assert.Empty(t, waiting.DecidedByName)
	assert.Empty(t, waiting.DecisionNote)
	assert.Empty(t, waiting.PreviewDigest)
	assert.Nil(t, waiting.CommitsAt)
	assert.Nil(t, waiting.UndoneAt)

	require.Len(t, users.asked, 1, "the deciders are read in one request")
	assert.Equal(t, []pulid.ID{fx.deciderID}, users.asked[0])
}

func TestListThreadPlans_ADeciderWhoCannotBeReadLeavesTheNameEmpty(t *testing.T) {
	t.Parallel()

	fx := newDecidedPlanFixture(&namedUsers{err: errors.New("users unavailable")})

	listed, err := fx.svc.ListThreadPlans(
		t.Context(),
		conversationports.GetThreadRequest{ID: fx.thread.ID},
	)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, fx.deciderID, listed[0].DecidedByUserID)
	assert.Empty(t, listed[0].DecidedByName)
}

func TestListThreadPlans_ADeciderOutsideTheOrganizationIsNotNamed(t *testing.T) {
	t.Parallel()

	fx := newDecidedPlanFixture(&namedUsers{})

	listed, err := fx.svc.ListThreadPlans(
		t.Context(),
		conversationports.GetThreadRequest{ID: fx.thread.ID},
	)
	require.NoError(t, err)
	assert.Empty(t, listed[0].DecidedByName)
}
