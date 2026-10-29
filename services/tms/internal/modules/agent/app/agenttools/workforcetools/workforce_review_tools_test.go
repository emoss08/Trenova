package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReviews struct {
	ReviewKeeper

	guard  *agenttoolstest.WriteGuard
	review *workerreview.PerformanceReview

	created *performancereviewservice.CreateReviewRequest
	saved   *performancereviewservice.UpdateReviewRequest
	deleted []pulid.ID
}

func newFakeReviews() *fakeReviews {
	three := int32(3)
	return &fakeReviews{
		guard: &agenttoolstest.WriteGuard{},
		review: &workerreview.PerformanceReview{
			ID:       pulid.MustNew("prv_"),
			WorkerID: pulid.MustNew("wrk_"),
			Status:   workerreview.ReviewStatusDraft,
			Title:    "2026 annual review",
			Summary:  "Steady year",
			Ratings: []workerreview.ReviewRating{
				{Key: "safety", Label: "Safety", Weight: 2, Score: &three},
				{Key: "service", Label: "Customer service", Weight: 1},
			},
			Goals:   []workerreview.ReviewGoal{{ID: "g1", Title: "Zero preventables"}},
			Version: 5,
		},
	}
}

func (f *fakeReviews) GetReview(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerreview.PerformanceReview, error) {
	copied := *f.review
	return &copied, nil
}

func (f *fakeReviews) PlanCreateReview(
	_ context.Context,
	req *performancereviewservice.CreateReviewRequest,
) (*workerreview.PerformanceReview, error) {
	return &workerreview.PerformanceReview{
		WorkerID:    req.WorkerID,
		TemplateID:  req.TemplateID,
		Status:      workerreview.ReviewStatusDraft,
		Title:       "Annual review",
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
	}, nil
}

func (f *fakeReviews) CreateReview(
	_ context.Context,
	req *performancereviewservice.CreateReviewRequest,
) (*workerreview.PerformanceReview, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.created = req
	return f.review, nil
}

func (f *fakeReviews) PlanUpdateReview(
	_ context.Context,
	req *performancereviewservice.UpdateReviewRequest,
) (*performancereviewservice.ReviewChange, error) {
	if f.review.Status != workerreview.ReviewStatusDraft {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"Only a draft review can be edited")
	}
	after := *f.review
	after.Ratings = req.Ratings
	after.Summary = req.Summary
	after.Goals = req.Goals
	return &performancereviewservice.ReviewChange{Before: f.review, After: &after}, nil
}

func (f *fakeReviews) UpdateReview(
	_ context.Context,
	req *performancereviewservice.UpdateReviewRequest,
) (*workerreview.PerformanceReview, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.saved = req
	return f.review, nil
}

func (f *fakeReviews) PlanDeleteReview(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerreview.PerformanceReview, error) {
	if f.review.Status != workerreview.ReviewStatusDraft {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"Only a draft review can be deleted")
	}
	return f.review, nil
}

func (f *fakeReviews) DeleteReview(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestStartPerformanceReview_OpensADraft(t *testing.T) {
	t.Parallel()

	reviews := newFakeReviews()
	tool := newStartPerformanceReviewTool(reviews)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:       reviews.review.WorkerID.String(),
		paramReviewTemplate:            pulid.MustNew("prt_").String(),
		changepreview.FieldPeriodStart: "2026-01-01",
		changepreview.FieldPeriodEnd:   "2026-12-31",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, reviews.created)
	assert.Equal(t, int64(1_767_225_600), reviews.created.PeriodStart)
	assert.Equal(t, permission.ResourcePerformanceReview, tool.Policy().Resource)
}

func TestDraftPerformanceReview_MergesRatingsByKey(t *testing.T) {
	t.Parallel()

	reviews := newFakeReviews()
	tool := newDraftPerformanceReviewTool(reviews)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramReviewID: reviews.review.ID.String(),
		paramRatings: []any{map[string]any{
			paramRatingKey:     "service",
			paramRatingScore:   float64(4),
			paramRatingComment: "Two customer compliments on file",
		}},
		paramGoals: []any{map[string]any{
			paramGoalTitle: "Finish hazmat training",
			"dueAt":        "2027-03-31",
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "2026 annual review")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, reviews.saved)
	require.Len(t, reviews.saved.Ratings, 2)
	require.NotNil(t, reviews.saved.Ratings[0].Score)
	assert.Equal(t, int32(3), *reviews.saved.Ratings[0].Score, "safety keeps its score")
	require.NotNil(t, reviews.saved.Ratings[1].Score)
	assert.Equal(t, int32(4), *reviews.saved.Ratings[1].Score)
	assert.Equal(t, "Steady year", reviews.saved.Summary)
	require.Len(t, reviews.saved.Goals, 1)
	assert.Equal(t, workerreview.ReviewGoalStatusOpen, reviews.saved.Goals[0].Status)
	require.NotNil(t, reviews.saved.Goals[0].DueAt)
	assert.Equal(t, reviews.review.Version, reviews.saved.Version)

	for name, raw := range map[string]map[string]any{
		"unknown key": {
			paramReviewID: reviews.review.ID.String(),
			paramRatings:  []any{map[string]any{paramRatingKey: "punctuality"}},
		},
		"score out of range": {
			paramReviewID: reviews.review.ID.String(),
			paramRatings: []any{map[string]any{
				paramRatingKey:   "safety",
				paramRatingScore: float64(9),
			}},
		},
		"unknown goal status": {
			paramReviewID: reviews.review.ID.String(),
			paramGoals: []any{map[string]any{
				paramGoalTitle:  "x",
				paramGoalStatus: "Abandoned",
			}},
		},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}

	reviews.review.Status = workerreview.ReviewStatusSubmitted
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params),
		"a submitted review is the reviewer's")
}

func TestDeletePerformanceReview_OnlyADraft(t *testing.T) {
	t.Parallel()

	reviews := newFakeReviews()
	tool := newDeletePerformanceReviewTool(reviews)
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramReviewID: reviews.review.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{reviews.review.ID}, reviews.deleted)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	reviews.review.Status = workerreview.ReviewStatusClosed
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}
