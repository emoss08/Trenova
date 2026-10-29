package agentplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/decisioncommitjobs/decisioncommitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideOwn_AStalePlanIsRefusedAtOnceRatherThanHeldForItsWindow(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.svc.previews = &fakePreviews{stale: true, staleStep: 1}
	window := &unreachableWindow{}
	f.svc.workflows = window
	followUps := &recordedFollowUps{}
	f.svc.followUps = followUps

	plan, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)
	require.NoError(t, err)

	assert.Equal(t, agentplan.PlanStatusFailed, plan.Status)
	assert.True(t, agentplan.PlanTargetChanged(plan), "its decider is told it was refused")
	assert.Empty(t, window.started, "nothing is left waiting on an undo window")
	require.Len(t, f.decider.decided, 1)
	assert.Equal(t, f.steps.steps[0].ID, f.decider.decided[0].ProposalID)
	require.Len(t, followUps.requests, 1)
	assert.Equal(t, plan.ID, followUps.requests[0].PlanID)

	require.NoError(
		t,
		f.svc.CommitOwnApprovalNow(t.Context(), &agentapprovalports.SettleApprovalRequest{
			PlanID:     plan.ID,
			TenantInfo: f.req.TenantInfo,
		}, f.actor),
		"committing a refused plan now leaves it as it is",
	)
	assert.Len(t, f.decider.decided, 1)
	assert.Len(t, followUps.requests, 1)
}

func TestCommitApproval_APlanWhoseRecordMovedInItsWindowIsRefusedAndReportedOnce(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.svc.workflows = &unreachableWindow{}
	previews := &fakePreviews{}
	f.svc.previews = previews
	followUps := &recordedFollowUps{}
	f.svc.followUps = followUps
	announced := &announcedPlans{}
	f.svc.activity = announced

	plan, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)
	require.NoError(t, err)
	require.Equal(t, agentplan.PlanStatusApproving, plan.Status)
	require.NotNil(t, plan.CommitsAt)

	previews.stale = true
	previews.staleStep = 2
	commit := &agentapprovalports.CommitApprovalRequest{
		OrganizationID:  f.req.TenantInfo.OrgID,
		BusinessUnitID:  f.req.TenantInfo.BuID,
		DecidedByUserID: f.actor.UserID,
		WorkflowID:      decisioncommitcontract.WorkflowID(plan.ID),
		PlanID:          plan.ID,
		CommitsAt:       *plan.CommitsAt,
	}
	require.NoError(t, f.svc.CommitApproval(t.Context(), commit))

	assert.Equal(t, agentplan.PlanStatusFailed, f.plans.plan.Status,
		"the plan is closed, not sent back to waiting")
	require.NotNil(t, f.plans.plan.FailedStep)
	assert.Equal(t, 2, *f.plans.plan.FailedStep)
	require.Len(t, f.decider.decided, 1, "only the moved step is recorded, and it runs nothing")
	assert.Equal(t, f.steps.steps[1].ID, f.decider.decided[0].ProposalID)
	require.Len(t, followUps.requests, 1)
	assert.NotContains(t, announced.actions, agentactivityports.ActivityReopened)

	require.NoError(t, f.svc.CommitApproval(t.Context(), commit),
		"a retried commit activity finds the plan already settled")
	assert.Len(t, f.decider.decided, 1)
	assert.Len(t, followUps.requests, 1, "a retry starts no second follow-up")
}
