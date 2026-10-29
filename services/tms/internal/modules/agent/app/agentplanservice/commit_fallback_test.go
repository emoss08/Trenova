package agentplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

type announcedPlans struct {
	agentactivityports.AgentActivityPublisher

	actions []string
}

func (a *announcedPlans) PlanChanged(
	_ context.Context,
	_ *agentplan.AgentPlan,
	_ kernelports.AuditActor,
	action string,
) {
	a.actions = append(a.actions, action)
}

type unreachableWindow struct {
	kernelports.WorkflowStarter

	started []*agentapprovalports.CommitApprovalRequest
}

func (w *unreachableWindow) Enabled() bool { return true }

func (w *unreachableWindow) StartWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	if len(args) == 1 {
		if req, ok := args[0].(*agentapprovalports.CommitApprovalRequest); ok {
			w.started = append(w.started, req)
		}
	}

	return nil, nil
}

func (w *unreachableWindow) SignalWorkflow(context.Context, string, string, string, any) error {
	return errors.New("workflow not found")
}

func gatedOwnFixture(t *testing.T) (*ownFixture, *unreachableWindow) {
	t.Helper()

	f := newOwnFixture(t)
	for _, step := range f.steps.steps {
		step.EgressClass = agent.EgressMoney
	}
	f.svc.previews = &fakePreviews{}
	window := &unreachableWindow{}
	f.svc.workflows = window

	return f, window
}

/*
Commit-now that cannot reach the plan's workflow commits the plan itself. The
approval it carries is the one the person made: the same reason, note and
preview digest the workflow would have been handed, so a plan whose steps must
be reviewed still goes through as approved.
*/
func TestCommitOwnApprovalNow_WithoutItsWorkflowCommitsTheApprovalThePersonMade(t *testing.T) {
	t.Parallel()

	f, window := gatedOwnFixture(t)
	f.req.PreviewDigest = "plan-digest"
	f.req.Note = "Checked against the remittance"

	plan, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)
	require.NoError(t, err)
	require.Equal(t, agentplan.PlanStatusApproving, plan.Status)
	require.Len(t, window.started, 1)
	assert.Equal(t, "plan-digest", window.started[0].PreviewDigest)
	assert.Equal(
		t,
		"plan-digest",
		f.plans.plan.ApprovalPreviewDigest,
		"the approval is kept with the plan",
	)

	require.NoError(
		t,
		f.svc.CommitOwnApprovalNow(t.Context(), &agentapprovalports.SettleApprovalRequest{
			PlanID:     plan.ID,
			TenantInfo: f.req.TenantInfo,
		}, f.actor),
	)

	assert.Equal(t, agentplan.PlanStatusCompleted, f.plans.plan.Status)
	require.Len(t, f.decider.decided, len(f.steps.steps))
	for _, decided := range f.decider.decided {
		assert.True(t, decided.StepPreviewReviewed, "each step went through as reviewed")
		assert.Contains(t, decided.ReasonCode, f.req.ReasonCode)
		assert.Equal(t, "Checked against the remittance", decided.Note)
	}
}

/*
A plan that reaches its commit without the preview its steps need is not
reported as committed: it goes back to waiting on the person and the caller
hears why.
*/
func TestCommitApproval_AGatedPlanWithoutItsPreviewIsNotReportedAsCommitted(t *testing.T) {
	t.Parallel()

	f, _ := gatedOwnFixture(t)
	announced := &announcedPlans{}
	f.svc.activity = announced
	f.req.PreviewDigest = ""
	commitsAt := int64(1_800_000_000)
	f.plans.plan.Status = agentplan.PlanStatusApproving
	f.plans.plan.CommitsAt = &commitsAt

	err := f.svc.CommitOwnApprovalNow(t.Context(), &agentapprovalports.SettleApprovalRequest{
		PlanID:     f.plans.plan.ID,
		TenantInfo: f.req.TenantInfo,
	}, f.actor)
	require.Error(t, err)
	assert.True(t, agent.IsReviewRequired(err))

	assert.Equal(
		t,
		agentplan.PlanStatusPending,
		f.plans.plan.Status,
		"the plan waits on the person again",
	)
	assert.Empty(t, f.decider.decided, "no step ran")
	assert.Equal(t, []string{agentactivityports.ActivityReopened}, announced.actions,
		"nobody undid it, so it is not announced as undone")
}
