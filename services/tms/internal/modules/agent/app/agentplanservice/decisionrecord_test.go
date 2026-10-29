package agentplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
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
	assert.Equal(t, agentplanports.PlanApproval{
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

func TestDecide_ARejectionRecordsItsNoteAndNoDigestEvenWhenOneIsSent(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.svc.previews = &fakePreviews{}
	h.req.Decision = agentdecision.DecisionRejected
	h.req.Note = "Hold off. I'm calling Tyson myself."
	h.req.PreviewDigest = "a-digest-nobody-checked"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, h.plans.statuses, 1)
	assert.Equal(t, agentplan.PlanStatusRejected, h.plans.statuses[0].Status)
	assert.Equal(t, "Hold off. I'm calling Tyson myself.", h.plans.statuses[0].Approval.Note)
	assert.Empty(t, h.plans.statuses[0].Approval.PreviewDigest)
}

func TestDecide_AnApprovalWithNoPreviewToSettleRecordsNoDigest(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.req.PreviewDigest = "a-digest-nobody-checked"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.NotEmpty(t, h.plans.statuses)
	assert.Empty(t, h.plans.statuses[0].Approval.PreviewDigest)
}

func TestCommitApproval_KeepsTheDigestStoredAtApprovalWhenNoPreviewSettlesAtCommit(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	commitsAt := int64(1_800_000_000)
	h.plans.plan.Status = agentplan.PlanStatusApproving
	h.plans.plan.CommitsAt = &commitsAt
	h.plans.plan.ApprovalPreviewDigest = "sha256:stored-at-approval"

	err := h.svc.CommitApproval(t.Context(), &agentapprovalports.CommitApprovalRequest{
		OrganizationID:  h.req.TenantInfo.OrgID,
		BusinessUnitID:  h.req.TenantInfo.BuID,
		DecidedByUserID: h.actor.UserID,
		PlanID:          h.plans.plan.ID,
		CommitsAt:       commitsAt,
	})
	require.NoError(t, err)

	require.NotEmpty(t, h.plans.statuses)
	assert.Equal(t, "sha256:stored-at-approval", h.plans.statuses[0].Approval.PreviewDigest)
}
