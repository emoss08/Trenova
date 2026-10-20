package agentplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecide_AnApprovalRecordsItsNoteAndTheDigestItWasMadeOn(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{}
	h.req.PreviewDigest = "plan-digest"
	h.req.Note = "  Go ahead, Samuel is already at the yard.  "

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.NotEmpty(t, h.plans.statuses)
	assert.Equal(t, agentports.PlanApproval{
		ReasonCode:    "looks right",
		Note:          "Go ahead, Samuel is already at the yard.",
		PreviewDigest: "plan-digest",
	}, h.plans.statuses[0].Approval)
	assert.Equal(t, h.actor.UserID, h.plans.statuses[0].DecidedByUserID)
}

func TestDecide_AnApprovalThatNamedNoDigestRecordsTheOneItWasDecidedOn(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{}

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.NotEmpty(t, h.plans.statuses)
	assert.Equal(t, "plan-digest", h.plans.statuses[0].Approval.PreviewDigest)
}

func TestDecide_ARejectionRecordsItsNoteAndNoDigest(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{}
	h.req.Decision = agent.DecisionRejected
	h.req.Note = "Hold off. I'm calling Tyson myself."

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, h.plans.statuses, 1)
	assert.Equal(t, agent.PlanStatusRejected, h.plans.statuses[0].Status)
	assert.Equal(t, "Hold off. I'm calling Tyson myself.", h.plans.statuses[0].Approval.Note)
	assert.Empty(t, h.plans.statuses[0].Approval.PreviewDigest)
}
