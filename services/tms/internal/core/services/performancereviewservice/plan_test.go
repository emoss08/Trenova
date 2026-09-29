package performancereviewservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanReview_DraftsWithoutSaving(t *testing.T) {
	h := newHarness(t)
	now := timeutils.NowUnix()

	planned, err := h.svc.PlanCreateReview(t.Context(),
		&performancereviewservice.CreateReviewRequest{
			TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID,
			PeriodStart: now - 180*86400, PeriodEnd: now, UserID: h.userID,
		})
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusDraft, planned.Status)
	assert.Len(t, planned.Ratings, 2)
	assert.Equal(t, h.template.Name, planned.Template.Name)
	assert.Empty(t, h.repo.reviews)

	draft, err := h.svc.CreateReview(t.Context(), &performancereviewservice.CreateReviewRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID,
		PeriodStart: now - 180*86400, PeriodEnd: now, UserID: h.userID,
	})
	require.NoError(t, err)

	four := int32(4)
	change, err := h.svc.PlanUpdateReview(t.Context(),
		&performancereviewservice.UpdateReviewRequest{
			TenantInfo: h.tenant,
			ID:         draft.ID,
			Ratings:    []worker.ReviewRating{{Key: "safety", Score: &four}},
			Summary:    " Solid year ",
		})
	require.NoError(t, err)
	assert.Equal(t, "Solid year", change.After.Summary)
	require.NotNil(t, change.After.Ratings[0].Score)
	assert.Nil(t, h.repo.reviews[0].Ratings[0].Score)

	gone, err := h.svc.PlanDeleteReview(t.Context(), h.tenant, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, draft.ID, gone.ID)
	assert.Len(t, h.repo.reviews, 1)
}
