package agentplanservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePlans struct {
	agentplanports.AgentPlanRepository

	plan     *agentplan.AgentPlan
	statuses []agentplanports.UpdateAgentPlanStatusRequest
	progress []agentplanports.RecordAgentPlanProgressRequest
}

func (f *fakePlans) GetByID(
	context.Context,
	agentplanports.GetAgentPlanByIDRequest,
) (*agentplan.AgentPlan, error) {
	return f.plan, nil
}

func (f *fakePlans) UpdateStatus(
	_ context.Context,
	req *agentplanports.UpdateAgentPlanStatusRequest,
) (*agentplan.AgentPlan, error) {
	f.statuses = append(f.statuses, *req)
	f.plan.Status = req.Status
	if req.CommitsAt != 0 {
		commitsAt := req.CommitsAt
		f.plan.CommitsAt = &commitsAt
		f.plan.ApprovalReasonCode = req.Approval.ReasonCode
		f.plan.ApprovalNote = req.Approval.Note
		f.plan.ApprovalPreviewDigest = req.Approval.PreviewDigest
	}
	if req.Reopen {
		f.plan.CommitsAt = nil
		f.plan.ApprovalReasonCode, f.plan.ApprovalNote, f.plan.ApprovalPreviewDigest = "", "", ""
	}

	return f.plan, nil
}

func (f *fakePlans) RecordProgress(
	_ context.Context,
	req *agentplanports.RecordAgentPlanProgressRequest,
) (*agentplan.AgentPlan, error) {
	f.progress = append(f.progress, *req)
	f.plan.Status = req.Status
	f.plan.CompletedSteps = req.CompletedSteps
	f.plan.FailedStep = req.FailedStep
	f.plan.FailureError = req.FailureError

	return f.plan, nil
}

type fakeSteps struct {
	agentproposalports.AgentProposalRepository

	steps   []*agent.AgentProposal
	skipped int
}

func (f *fakeSteps) ListByPlan(
	context.Context,
	agentproposalports.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return f.steps, nil
}

func (f *fakeSteps) SkipPendingByPlan(
	context.Context,
	agentproposalports.SkipPendingByPlanRequest,
) (int, error) {
	f.skipped++

	return 1, nil
}

type fakeDecider struct {
	failAt   pulid.ID
	decided  []agentdecisionports.DecideAgentProposalRequest
	signals  int
	lastSent *agentdecisionports.DecideAgentProposalRequest
}

func (f *fakeDecider) DecideWithOutcome(
	_ context.Context,
	req *agentdecisionports.DecideAgentProposalRequest,
	_ *permissionports.RequestActor,
) (*agentdecisionports.DecisionOutcome, error) {
	f.decided = append(f.decided, *req)
	outcome := &agentdecisionports.DecisionOutcome{
		Decision: &agentdecision.AgentDecision{ID: pulid.MustNew("ad_")},
	}
	if req.ProposalID == f.failAt {
		outcome.ExecutionError = errors.New("the record changed since this was proposed")
	}
	if req.StepPreview.IsStale() {
		outcome.ExecutionError = fmt.Errorf("%w: the shipment moved on", agentplan.ErrTargetChanged)
	}

	return outcome, nil
}

func (f *fakeDecider) SignalRun(
	_ context.Context,
	req *agentdecisionports.DecideAgentProposalRequest,
	_ *agentdecision.AgentDecision,
	_ pulid.ID,
) error {
	f.signals++
	f.lastSent = req

	return nil
}

type fakeShadow struct {
	shadow bool
	runs   map[pulid.ID]bool
}

func (f fakeShadow) ForRun(
	_ context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) (agentshadow.Verdict, error) {
	if f.shadow {
		return agentshadow.Verdict{Cause: agentshadow.CauseOrganization}, nil
	}
	if f.runs[runID] {
		return agentshadow.Verdict{Cause: agentshadow.CauseDefinition, AgentName: "Shipment Desk"},
			nil
	}

	return agentshadow.Verdict{}, nil
}

func (f fakeShadow) ForRuns(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runIDs []pulid.ID,
) (map[pulid.ID]agentshadow.Verdict, error) {
	verdicts := make(map[pulid.ID]agentshadow.Verdict, len(runIDs))
	for _, runID := range runIDs {
		verdicts[runID], _ = f.ForRun(ctx, tenant, runID)
	}

	return verdicts, nil
}

type harness struct {
	svc     *Service
	plans   *fakePlans
	steps   *fakeSteps
	decider *fakeDecider
	actor   *permissionports.RequestActor
	req     *agentplanports.DecideAgentPlanRequest
}

func newHarness(t *testing.T, stepCount int, shadow bool) *harness {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	runID := pulid.MustNew("arun_")
	plan := &agentplan.AgentPlan{
		ID: pulid.MustNew("apl_"), OrganizationID: orgID, BusinessUnitID: buID, RunID: runID,
		Title: "Dispatch coverage: 3 changes", Status: agentplan.PlanStatusPending, StepCount: stepCount,
	}
	steps := make([]*agent.AgentProposal, 0, stepCount)
	for i := 1; i <= stepCount; i++ {
		steps = append(steps, &agent.AgentProposal{
			ID: pulid.MustNew("ap_"), OrganizationID: orgID, BusinessUnitID: buID, RunID: runID,
			ToolName: "dispatch_assign_move", Status: agent.ProposalStatusPending, PlanStep: i,
			EgressClass: agent.EgressInternal,
		})
	}

	h := &harness{
		plans:   &fakePlans{plan: plan},
		steps:   &fakeSteps{steps: steps},
		decider: &fakeDecider{},
		actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser, PrincipalID: pulid.MustNew("usr_"),
			UserID: pulid.MustNew("usr_"), OrganizationID: orgID, BusinessUnitID: buID,
		},
		req: &agentplanports.DecideAgentPlanRequest{
			PlanID: plan.ID, Decision: agentdecision.DecisionAccepted, ReasonCode: "looks right",
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
	}
	h.svc = &Service{
		l: zap.NewNop(), plans: h.plans, proposals: h.steps, decisions: h.decider, shadow: fakeShadow{shadow: shadow},
	}

	return h
}

func TestDecide_ApprovesEveryStepInOrderAndSignalsOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agentplan.PlanStatusCompleted, plan.Status)
	assert.Equal(t, 3, plan.CompletedSteps)

	require.Len(t, h.decider.decided, 3)
	for i, decided := range h.decider.decided {
		assert.Equal(t, h.steps.steps[i].ID, decided.ProposalID, "steps run in plan order")
		assert.Equal(t, agentdecision.DecisionAccepted, decided.Decision)
		assert.True(t, decided.WithinPlan)
		assert.Contains(t, decided.ReasonCode, "looks right (plan ")
	}
	assert.Equal(t, 1, h.decider.signals, "the run hears once, not once per step")

	require.Len(t, h.plans.statuses, 1)
	assert.Equal(t, agentplan.PlanStatusApproved, h.plans.statuses[0].Status)
	assert.Equal(
		t,
		agentplan.PlanStatusPending,
		h.plans.statuses[0].FromStatus,
		"the claim is conditional",
	)
	assert.Equal(t, h.actor.UserID, h.plans.statuses[0].DecidedByUserID)
}

func TestDecide_StopsAtTheFirstFailedStepAndSkipsTheRest(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	h.decider.failAt = h.steps.steps[1].ID

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err, "a failed step is the plan's outcome, not an error to the caller")

	assert.Equal(t, agentplan.PlanStatusFailed, plan.Status)
	assert.Equal(t, 1, plan.CompletedSteps)
	require.NotNil(t, plan.FailedStep)
	assert.Equal(t, 2, *plan.FailedStep)
	assert.Contains(t, plan.FailureError, "changed since")

	assert.Len(t, h.decider.decided, 2, "the third step is never decided")
	assert.Equal(t, 1, h.steps.skipped, "the steps after the failure are skipped")
	assert.Equal(t, 0, h.decider.signals)
}

func TestDecide_RejectsEveryPendingStep(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.steps.steps[0].Status = agent.ProposalStatusExecuted
	h.req.Decision = agentdecision.DecisionRejected

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agentplan.PlanStatusRejected, plan.Status)
	require.Len(t, h.decider.decided, 1, "a step already decided on its own is left alone")
	assert.Equal(t, h.steps.steps[1].ID, h.decider.decided[0].ProposalID)
	assert.Equal(t, agentdecision.DecisionRejected, h.decider.decided[0].Decision)
	assert.Equal(t, 1, h.decider.signals)
}

func TestDecide_RefusesWhatCannotBeDecided(t *testing.T) {
	t.Parallel()

	shadowed := newHarness(t, 2, true)
	_, err := shadowed.svc.Decide(t.Context(), shadowed.req, shadowed.actor)
	require.ErrorContains(t, err, "paused")
	assert.Empty(t, shadowed.decider.decided)

	decided := newHarness(t, 2, false)
	decided.plans.plan.Status = agentplan.PlanStatusCompleted
	_, err = decided.svc.Decide(t.Context(), decided.req, decided.actor)
	require.ErrorContains(t, err, "already been decided")

	modified := newHarness(t, 2, false)
	modified.req.Decision = agentdecision.DecisionModified
	_, err = modified.svc.Decide(t.Context(), modified.req, modified.actor)
	require.ErrorContains(t, err, "as a whole")

	agentActor := newHarness(t, 2, false)
	agentActor.actor.PrincipalType = permissionports.PrincipalTypeAgent
	agentActor.actor.UserID = pulid.Nil
	_, err = agentActor.svc.Decide(t.Context(), agentActor.req, agentActor.actor)
	require.ErrorContains(t, err, "human")
}

// A step filed by a delegate whose shadow switch is on holds the whole plan:
// approving it would run a write the switch exists to withhold.
func TestDecide_RefusesAPlanWithAStepFromAnAgentInShadowMode(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	delegateRun := pulid.MustNew("arun_")
	h.steps.steps[1].RunID = delegateRun
	h.svc.shadow = fakeShadow{runs: map[pulid.ID]bool{delegateRun: true}}

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	require.ErrorContains(t, err, "Shipment Desk")
	assert.Empty(t, h.decider.decided)
	assert.Empty(t, h.plans.statuses)
}

func (f *fakeSteps) SumAutomaticMoney(
	context.Context,
	*agentproposalports.SumAutomaticMoneyRequest,
) (agenttooltrust.MoneyUse, error) {
	return agenttooltrust.MoneyUse{}, nil
}

func TestDecide_AGatedPlanApprovedUnseenIsRefusedBeforeAnyStep(t *testing.T) {
	t.Parallel()

	for name, gate := range map[string]func(*agent.AgentProposal){
		"money":   func(step *agent.AgentProposal) { step.EgressClass = agent.EgressMoney },
		"tainted": func(step *agent.AgentProposal) { step.Tainted = true },
		"unknown": func(step *agent.AgentProposal) { step.EgressClass = "" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, 3, false)
			gate(h.steps.steps[1])

			_, err := h.svc.Decide(t.Context(), h.req, h.actor)

			require.Error(t, err)
			assert.True(t, agent.IsReviewRequired(err))
			assert.Empty(t, h.decider.decided, "no step is decided")
			assert.Empty(t, h.plans.statuses, "the plan is not claimed")
		})
	}
}

func TestDecide_AGatedPlanCanAlwaysBeRejectedUnseen(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.steps.steps[0].EgressClass = agent.EgressExternalRecipient
	h.req.Decision = agentdecision.DecisionRejected

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)
}

func (f *fakePlans) ListConnection(
	context.Context,
	*agentplanports.ListAgentPlanConnectionRequest,
) (*pagination.CursorListResult[*agentplan.AgentPlan], error) {
	return &pagination.CursorListResult[*agentplan.AgentPlan]{
		Items: []*agentplan.AgentPlan{f.plan},
	}, nil
}

/*
AI Control lists every plan, those a switch holds included, and opens one by
its id: a held plan says it is held where its buttons would be, rather than
vanishing while it waits. Only a decision on it is refused.
*/
func TestListConnectionAndGetByID_KeepAPlanTheOrganizationPauseHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, true)

	listed, err := h.svc.ListConnection(t.Context(), &agentplanports.ListAgentPlanConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: h.req.TenantInfo},
	})
	require.NoError(t, err)
	require.Equal(t, []*agentplan.AgentPlan{h.plans.plan}, listed.Items)

	found, err := h.svc.GetByID(t.Context(), agentplanports.GetAgentPlanByIDRequest{
		ID:         h.plans.plan.ID,
		TenantInfo: h.req.TenantInfo,
	})
	require.NoError(t, err)
	require.Equal(t, h.plans.plan, found)

	_, err = h.svc.Decide(t.Context(), h.req, h.actor)
	require.Error(t, err, "a decision on a held plan is still refused")
	assert.Equal(
		t,
		shadowRefusal(agentshadow.Verdict{Cause: agentshadow.CauseOrganization}).Error(),
		err.Error(),
	)
	assert.Empty(t, h.plans.statuses, "a held plan is not moved out of Pending")
	assert.Empty(t, h.decider.decided, "no step of a held plan is decided")
}

func (f *fakeSteps) ListByPlans(
	context.Context,
	agentproposalports.ListAgentProposalsByPlansRequest,
) ([]*agent.AgentProposal, error) {
	return f.steps, nil
}

/*
A plan read on AI Control or the Desk says it is held by the same rule a
decision on it is refused by: a step a delegate raised from an agent in
shadow holds the whole plan, though the plan's own run is free.
*/
func TestPlanHolds_HoldsAPlanByAStepAnotherRunRaisedAsDecideRefusesIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	delegateRun := pulid.MustNew("arun_")
	planID := h.plans.plan.ID
	h.steps.steps[1].RunID = delegateRun
	h.steps.steps[0].PlanID = &planID
	h.steps.steps[1].PlanID = &planID
	h.svc.shadow = fakeShadow{runs: map[pulid.ID]bool{delegateRun: true}}

	holds, err := h.svc.PlanHolds(
		t.Context(),
		h.req.TenantInfo,
		[]*agentplan.AgentPlan{h.plans.plan},
	)
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]*assistantproposalports.ProposalHold{
		planID: {Reason: string(agentshadow.CauseDefinition), AgentName: "Shipment Desk"},
	}, holds)

	_, err = h.svc.Decide(t.Context(), h.req, h.actor)
	require.ErrorContains(t, err, "Shipment Desk")
}

func TestPlanHolds_LeavesADecidedPlanAndAFreeOneUnheld(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1, true)
	h.plans.plan.Status = agentplan.PlanStatusRejected

	holds, err := h.svc.PlanHolds(
		t.Context(),
		h.req.TenantInfo,
		[]*agentplan.AgentPlan{h.plans.plan},
	)
	require.NoError(t, err)
	assert.Empty(t, holds)

	free := newHarness(t, 1, false)
	holds, err = free.svc.PlanHolds(
		t.Context(),
		free.req.TenantInfo,
		[]*agentplan.AgentPlan{free.plans.plan},
	)
	require.NoError(t, err)
	assert.Empty(t, holds)
}
