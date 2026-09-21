package agentplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePlans struct {
	repositories.AgentPlanRepository

	plan     *agent.AgentPlan
	statuses []repositories.UpdateAgentPlanStatusRequest
	progress []repositories.RecordAgentPlanProgressRequest
}

func (f *fakePlans) GetByID(context.Context, repositories.GetAgentPlanByIDRequest) (*agent.AgentPlan, error) {
	return f.plan, nil
}

func (f *fakePlans) UpdateStatus(
	_ context.Context,
	req repositories.UpdateAgentPlanStatusRequest,
) (*agent.AgentPlan, error) {
	f.statuses = append(f.statuses, req)
	f.plan.Status = req.Status

	return f.plan, nil
}

func (f *fakePlans) RecordProgress(
	_ context.Context,
	req repositories.RecordAgentPlanProgressRequest,
) (*agent.AgentPlan, error) {
	f.progress = append(f.progress, req)
	f.plan.Status = req.Status
	f.plan.CompletedSteps = req.CompletedSteps
	f.plan.FailedStep = req.FailedStep
	f.plan.FailureError = req.FailureError

	return f.plan, nil
}

type fakeSteps struct {
	repositories.AgentProposalRepository

	steps   []*agent.AgentProposal
	skipped int
}

func (f *fakeSteps) ListByPlan(context.Context, repositories.ListAgentProposalsByPlanRequest) ([]*agent.AgentProposal, error) {
	return f.steps, nil
}

func (f *fakeSteps) SkipPendingByPlan(context.Context, repositories.SkipPendingByPlanRequest) (int, error) {
	f.skipped++

	return 1, nil
}

type fakeDecider struct {
	failAt   pulid.ID
	decided  []services.DecideAgentProposalRequest
	signals  int
	lastSent *services.DecideAgentProposalRequest
}

func (f *fakeDecider) DecideWithOutcome(
	_ context.Context,
	req *services.DecideAgentProposalRequest,
	_ *services.RequestActor,
) (*services.DecisionOutcome, error) {
	f.decided = append(f.decided, *req)
	outcome := &services.DecisionOutcome{Decision: &agent.AgentDecision{ID: pulid.MustNew("ad_")}}
	if req.ProposalID == f.failAt {
		outcome.ExecutionError = errors.New("the record changed since this was proposed")
	}

	return outcome, nil
}

func (f *fakeDecider) SignalRun(
	_ context.Context,
	req *services.DecideAgentProposalRequest,
	_ *agent.AgentDecision,
	_ pulid.ID,
) error {
	f.signals++
	f.lastSent = req

	return nil
}

type fakeShadow struct{ shadow bool }

func (f fakeShadow) Organization(context.Context, pagination.TenantInfo) (bool, error) {
	return f.shadow, nil
}

func (f fakeShadow) ForRun(context.Context, pagination.TenantInfo, pulid.ID) (agentshadow.Verdict, error) {
	if f.shadow {
		return agentshadow.Verdict{Cause: agentshadow.CauseOrganization}, nil
	}

	return agentshadow.Verdict{}, nil
}

type harness struct {
	svc     *Service
	plans   *fakePlans
	steps   *fakeSteps
	decider *fakeDecider
	actor   *services.RequestActor
	req     *services.DecideAgentPlanRequest
}

func newHarness(t *testing.T, stepCount int, shadow bool) *harness {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	runID := pulid.MustNew("arun_")
	plan := &agent.AgentPlan{
		ID: pulid.MustNew("apl_"), OrganizationID: orgID, BusinessUnitID: buID, RunID: runID,
		Title: "Dispatch coverage: 3 changes", Status: agent.PlanStatusPending, StepCount: stepCount,
	}
	steps := make([]*agent.AgentProposal, 0, stepCount)
	for i := 1; i <= stepCount; i++ {
		steps = append(steps, &agent.AgentProposal{
			ID: pulid.MustNew("ap_"), OrganizationID: orgID, BusinessUnitID: buID, RunID: runID,
			ToolName: "assign_move", Status: agent.ProposalStatusPending, PlanStep: i,
		})
	}

	h := &harness{
		plans:   &fakePlans{plan: plan},
		steps:   &fakeSteps{steps: steps},
		decider: &fakeDecider{},
		actor: &services.RequestActor{
			PrincipalType: services.PrincipalTypeUser, PrincipalID: pulid.MustNew("usr_"),
			UserID: pulid.MustNew("usr_"), OrganizationID: orgID, BusinessUnitID: buID,
		},
		req: &services.DecideAgentPlanRequest{
			PlanID: plan.ID, Decision: agent.DecisionAccepted, ReasonCode: "looks right",
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

	assert.Equal(t, agent.PlanStatusCompleted, plan.Status)
	assert.Equal(t, 3, plan.CompletedSteps)

	require.Len(t, h.decider.decided, 3)
	for i, decided := range h.decider.decided {
		assert.Equal(t, h.steps.steps[i].ID, decided.ProposalID, "steps run in plan order")
		assert.Equal(t, agent.DecisionAccepted, decided.Decision)
		assert.True(t, decided.WithinPlan)
		assert.Contains(t, decided.ReasonCode, "looks right (plan ")
	}
	assert.Equal(t, 1, h.decider.signals, "the run hears once, not once per step")

	require.Len(t, h.plans.statuses, 1)
	assert.Equal(t, agent.PlanStatusApproved, h.plans.statuses[0].Status)
	assert.Equal(t, agent.PlanStatusPending, h.plans.statuses[0].FromStatus, "the claim is conditional")
	assert.Equal(t, h.actor.UserID, h.plans.statuses[0].DecidedByUserID)
}

func TestDecide_StopsAtTheFirstFailedStepAndSkipsTheRest(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	h.decider.failAt = h.steps.steps[1].ID

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err, "a failed step is the plan's outcome, not an error to the caller")

	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
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
	h.req.Decision = agent.DecisionRejected

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agent.PlanStatusRejected, plan.Status)
	require.Len(t, h.decider.decided, 1, "a step already decided on its own is left alone")
	assert.Equal(t, h.steps.steps[1].ID, h.decider.decided[0].ProposalID)
	assert.Equal(t, agent.DecisionRejected, h.decider.decided[0].Decision)
	assert.Equal(t, 1, h.decider.signals)
}

func TestDecide_RefusesWhatCannotBeDecided(t *testing.T) {
	t.Parallel()

	shadowed := newHarness(t, 2, true)
	_, err := shadowed.svc.Decide(t.Context(), shadowed.req, shadowed.actor)
	require.ErrorContains(t, err, "paused")
	assert.Empty(t, shadowed.decider.decided)

	decided := newHarness(t, 2, false)
	decided.plans.plan.Status = agent.PlanStatusCompleted
	_, err = decided.svc.Decide(t.Context(), decided.req, decided.actor)
	require.ErrorContains(t, err, "already been decided")

	modified := newHarness(t, 2, false)
	modified.req.Decision = agent.DecisionModified
	_, err = modified.svc.Decide(t.Context(), modified.req, modified.actor)
	require.ErrorContains(t, err, "as a whole")

	agentActor := newHarness(t, 2, false)
	agentActor.actor.PrincipalType = services.PrincipalTypeAgent
	agentActor.actor.UserID = pulid.Nil
	_, err = agentActor.svc.Decide(t.Context(), agentActor.req, agentActor.actor)
	require.ErrorContains(t, err, "human")
}
