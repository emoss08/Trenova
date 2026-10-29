package performancereviewservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ReviewChange = kernelports.RecordChange[workerreview.PerformanceReview]

func (s *Service) PlanCreateReview(
	ctx context.Context,
	req *CreateReviewRequest,
) (*workerreview.PerformanceReview, error) {
	if _, err := s.loadWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
		return nil, err
	}
	template, err := s.repo.GetTemplateByID(ctx, &workerreviewports.GetReviewTemplateByIDRequest{
		ID:         req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if template.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"templateId",
			errortypes.ErrInvalidOperation,
			"Inactive templates cannot be used for a new review",
		)
	}

	entity := &workerreview.PerformanceReview{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		TemplateID:     req.TemplateID,
		ReviewerID:     req.UserID,
		Status:         workerreview.ReviewStatusDraft,
		Title:          strings.TrimSpace(req.Title),
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		Ratings:        workerreview.RatingsFromTemplate(template),
		Goals:          []workerreview.ReviewGoal{},
		Template:       template,
	}
	if entity.Title == "" {
		entity.Title = template.Name + " — " + timeutils.FormatUnixDateIn(req.PeriodEnd, "")
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return entity, nil
}

func (s *Service) PlanUpdateReview(
	ctx context.Context,
	req *UpdateReviewRequest,
) (*ReviewChange, error) {
	original, err := s.loadReview(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status != workerreview.ReviewStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only a draft can be edited. Reopen the review or start a new one",
		)
	}

	updated := *original
	if title := strings.TrimSpace(req.Title); title != "" {
		updated.Title = title
	}
	if req.PeriodStart > 0 {
		updated.PeriodStart = req.PeriodStart
	}
	if req.PeriodEnd > 0 {
		updated.PeriodEnd = req.PeriodEnd
	}
	updated.Ratings = mergeRatings(original.Ratings, req.Ratings)
	updated.OverallScore = workerreview.ComputeOverallScore(updated.Ratings)
	updated.Summary = strings.TrimSpace(req.Summary)
	updated.Strengths = strings.TrimSpace(req.Strengths)
	updated.Improvements = strings.TrimSpace(req.Improvements)
	updated.Goals = normalizeGoals(req.Goals)
	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &ReviewChange{Before: original, After: &updated}, nil
}

func (s *Service) PlanDeleteReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*workerreview.PerformanceReview, error) {
	original, err := s.loadReview(ctx, tenantInfo, id, 0)
	if err != nil {
		return nil, err
	}
	if original.Status != workerreview.ReviewStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only drafts can be deleted",
		)
	}
	return original, nil
}
