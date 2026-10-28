package agentplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePreviews struct {
	agentruntimeports.ProposalPreviewService

	stale     bool
	staleStep int
	asked     int
}

func (f *fakePreviews) ForPlan(
	_ context.Context,
	req *agentruntimeports.PlanPreviewRequest,
) (*agent.PlanPreview, error) {
	f.asked++
	preview := &agent.PlanPreview{PlanID: req.Plan.ID, Stale: f.stale, Digest: "plan-digest"}
	for _, step := range req.Steps {
		stepPreview := &agent.ProposalPreview{
			ProposalID: step.ID,
			Digest:     "step-" + step.ID.String(),
		}
		if f.stale && (f.staleStep == 0 || f.staleStep == step.PlanStep) {
			stepPreview.Staleness = &agent.PreviewStaleness{
				Pinned: true, ProposedVersion: 3, CurrentVersion: 4,
			}
		}
		preview.Steps = append(preview.Steps, agent.PlanStepPreview{
			ProposalID: step.ID,
			Step:       step.PlanStep,
			Preview:    stepPreview,
		})
	}

	return preview, nil
}

type recordedFollowUps struct {
	requests []assistantports.DecisionFollowUpRequest
}

func (r *recordedFollowUps) FollowUp(
	_ context.Context,
	req assistantports.DecisionFollowUpRequest,
) {
	r.requests = append(r.requests, req)
}

func TestDecide_AStalePlanIsRefusedAsApprovedAndNothingRuns(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	h.svc.previews = &fakePreviews{stale: true, staleStep: 2}
	followUps := &recordedFollowUps{}
	h.svc.followUps = followUps
	h.req.PreviewDigest = "the-digest-the-person-reviewed"

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err, "a refused plan is the plan's outcome, not an error to the caller")

	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
	assert.Equal(t, 0, plan.CompletedSteps)
	require.NotNil(t, plan.FailedStep)
	assert.Equal(t, 2, *plan.FailedStep)
	assert.Contains(t, plan.FailureError, "nothing ran")
	assert.Contains(t, plan.FailureError, "step 2: the record changed since this was proposed")
	assert.True(t, agent.PlanTargetChanged(plan),
		"the plan reads as refused over a changed record, not as a step that failed")

	require.Len(t, h.decider.decided, 1, "only the stale step is decided, and it runs nothing")
	refused := h.decider.decided[0]
	assert.Equal(t, h.steps.steps[1].ID, refused.ProposalID)
	assert.True(t, refused.WithinPlan)
	assert.True(
		t,
		refused.StepPreview.IsStale(),
		"the step is refused on the preview it moved on in",
	)
	assert.Equal(t, 1, h.steps.skipped, "the steps that were not stale are skipped, never run")

	require.Len(t, h.plans.statuses, 1)
	assert.Equal(t, agent.PlanStatusApproved, h.plans.statuses[0].Status)
	assert.Equal(t, agent.PlanStatusPending, h.plans.statuses[0].FromStatus,
		"the claim is conditional, so a second decision cannot refuse it again")
	assert.Equal(t, 1, h.decider.signals, "the run hears the plan is no longer waiting")
	require.Len(t, followUps.requests, 1, "the conversation is told why nothing ran")
	assert.Equal(t, plan.ID, followUps.requests[0].PlanID)
}

func TestDecide_AStalePlanDecidedAgainIsNotRefusedTwice(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{stale: true}
	followUps := &recordedFollowUps{}
	h.svc.followUps = followUps

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	_, err = h.svc.Decide(t.Context(), h.req, h.actor)
	require.ErrorContains(t, err, "already been decided")
	assert.Len(t, h.decider.decided, 2, "each stale step is refused once")
	assert.Len(t, followUps.requests, 1, "one decision, one follow-up")
}

func TestDecide_APlanDigestThatNoLongerMatchesIsAConflict(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{}
	h.req.PreviewDigest = "an-older-digest"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	var conflict *errortypes.ConflictError
	require.True(t, errors.As(err, &conflict), "%v", err)
	assert.Empty(t, h.plans.statuses)
	assert.Empty(t, h.decider.decided)
}

func TestDecide_EachStepRecordsThePreviewThePlanWasApprovedOn(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	previews := &fakePreviews{}
	h.svc.previews = previews
	h.req.PreviewDigest = "plan-digest"

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)
	assert.Equal(t, agent.PlanStatusCompleted, plan.Status)

	require.Len(t, h.decider.decided, 2)
	for i, decided := range h.decider.decided {
		require.NotNil(t, decided.StepPreview)
		assert.Equal(t, "step-"+h.steps.steps[i].ID.String(), decided.StepPreview.Digest)
		assert.True(t, decided.StepPreviewReviewed)
	}
	assert.Equal(t, 1, previews.asked)
}

func TestDecide_ARejectedPlanNeedsNoPreview(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	previews := &fakePreviews{stale: true}
	h.svc.previews = previews
	h.req.Decision = agent.DecisionRejected

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agent.PlanStatusRejected, plan.Status)
	assert.Zero(t, previews.asked)
}
