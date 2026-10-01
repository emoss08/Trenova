package agentplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func targetEach(h *harness, resource permission.Resource) {
	for _, step := range h.steps.steps {
		step.ToolName = "post_invoice"
		step.TargetResource = string(resource)
		step.TargetID = pulid.MustNew("inv_")
	}
}

func TestDecide_IndependentStepsAllRunAndEachOutcomeIsReported(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 5, false)
	targetEach(h, permission.ResourceInvoice)
	h.decider.failAt = h.steps.steps[1].ID

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, h.decider.decided, 5, "one bad invoice does not hold back the others")
	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
	assert.Equal(t, 4, plan.CompletedSteps)
	require.NotNil(t, plan.FailedStep)
	assert.Equal(t, 2, *plan.FailedStep)
	assert.Contains(t, plan.FailureError, "1 of 5 steps did not run")
	assert.Contains(t, plan.FailureError, "step 2")
	assert.Contains(t, plan.FailureError, "changed since")
	assert.Equal(t, 1, h.decider.signals, "the run hears once about what did run")
}

func TestDecide_IndependentStepsThatAllRunCompleteThePlan(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	targetEach(h, permission.ResourceInvoice)

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agent.PlanStatusCompleted, plan.Status)
	assert.Equal(t, 3, plan.CompletedSteps)
	assert.Zero(t, h.steps.skipped)
}

func TestDecide_StepsOnOneRecordStillStopAtTheFirstFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	targetEach(h, permission.ResourceInvoice)
	h.steps.steps[2].TargetID = h.steps.steps[0].TargetID
	h.decider.failAt = h.steps.steps[0].ID

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Len(t, h.decider.decided, 1, "a later step on the same record depends on the first")
	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
	assert.Equal(t, 1, h.steps.skipped)
}

func TestDecide_StepsAcrossResourcesStillStopAtTheFirstFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	targetEach(h, permission.ResourceInvoice)
	h.steps.steps[1].TargetResource = string(permission.ResourceBillingQueue)
	h.decider.failAt = h.steps.steps[0].ID

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Len(t, h.decider.decided, 1,
		"a step on another kind of record may rest on what an earlier step does")
}

func TestDecide_AnUntargetedStepKeepsThePlanInOrder(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	targetEach(h, permission.ResourceInvoice)
	h.steps.steps[2].TargetID = pulid.Nil
	h.decider.failAt = h.steps.steps[0].ID

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Len(t, h.decider.decided, 1)
}

func TestIndependent_ReadsTheStepsTargets(t *testing.T) {
	t.Parallel()

	shared := pulid.MustNew("inv_")
	steps := []*agent.AgentProposal{
		{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
		{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
	}
	assert.True(t, independent(steps))

	steps[1].TargetID = shared
	steps = append(steps, &agent.AgentProposal{TargetResource: "invoice", TargetID: shared})
	assert.False(t, independent(steps))

	assert.False(t, independent(steps[:1]), "one step has nothing to be independent of")
}
