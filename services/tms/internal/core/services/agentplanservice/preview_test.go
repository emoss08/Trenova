package agentplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePreviews struct {
	services.ProposalPreviewService

	stale bool
	asked int
}

func (f *fakePreviews) ForPlan(
	_ context.Context,
	req *services.PlanPreviewRequest,
) (*agent.PlanPreview, error) {
	f.asked++
	preview := &agent.PlanPreview{PlanID: req.Plan.ID, Stale: f.stale, Digest: "plan-digest"}
	for _, step := range req.Steps {
		preview.Steps = append(preview.Steps, agent.PlanStepPreview{
			ProposalID: step.ID,
			Step:       step.PlanStep,
			Preview:    &agent.ProposalPreview{ProposalID: step.ID, Digest: "step-" + step.ID.String()},
		})
	}

	return preview, nil
}

func TestDecide_AStalePlanIsRefusedBeforeAnythingRuns(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{stale: true}

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	var business *errortypes.BusinessError
	require.True(t, errors.As(err, &business), "%v", err)
	assert.Empty(t, h.plans.statuses, "the plan stays pending")
	assert.Empty(t, h.decider.decided)
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
