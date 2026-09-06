package performancereviewservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListReviews(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	statuses []worker.ReviewStatus,
) ([]*worker.PerformanceReview, error) {
	return s.repo.ListReviews(ctx, &repositories.ListPerformanceReviewsRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		Statuses:        statuses,
		IncludeTemplate: true,
		IncludeReviewer: true,
	})
}

func (s *Service) GetReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.PerformanceReview, error) {
	return s.repo.GetReviewByID(ctx, &repositories.GetPerformanceReviewByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeTemplate: true,
		IncludeWorker:   true,
		IncludeReviewer: true,
	})
}

type CreateReviewRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	TemplateID  pulid.ID
	Title       string
	PeriodStart int64
	PeriodEnd   int64
	UserID      pulid.ID
}

// CreateReview opens a draft with the template's items copied onto it and
// the caller as reviewer.
func (s *Service) CreateReview(
	ctx context.Context,
	req *CreateReviewRequest,
) (*worker.PerformanceReview, error) {
	log := s.l.With(zap.String("operation", "CreateReview"), zap.String("workerId", req.WorkerID.String()))

	if _, err := s.loadWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
		return nil, err
	}
	template, err := s.repo.GetTemplateByID(ctx, &repositories.GetReviewTemplateByIDRequest{
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

	entity := &worker.PerformanceReview{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		TemplateID:     req.TemplateID,
		ReviewerID:     req.UserID,
		Status:         worker.ReviewStatusDraft,
		Title:          strings.TrimSpace(req.Title),
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		Ratings:        worker.RatingsFromTemplate(template),
		Goals:          []worker.ReviewGoal{},
	}
	if entity.Title == "" {
		entity.Title = template.Name + " — " + timeutils.FormatUnixDateIn(req.PeriodEnd, "")
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateReview(ctx, entity)
	if err != nil {
		log.Error("failed to create review", zap.Error(err))
		return nil, err
	}
	created.Template = template

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: req.UserID, tenant: req.TenantInfo,
		current: created, comment: "Review drafted: " + created.Title, log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeReview, permission.OpCreate, created.ID, req.UserID)
	return created, nil
}

type UpdateReviewRequest struct {
	TenantInfo   pagination.TenantInfo
	ID           pulid.ID
	Title        string
	PeriodStart  int64
	PeriodEnd    int64
	Ratings      []worker.ReviewRating
	Summary      string
	Strengths    string
	Improvements string
	Goals        []worker.ReviewGoal
	Version      int64
	UserID       pulid.ID
}

// UpdateReview saves the reviewer's work on a draft. Ratings keep the
// template's keys, labels and weights; only scores and comments move.
func (s *Service) UpdateReview(
	ctx context.Context,
	req *UpdateReviewRequest,
) (*worker.PerformanceReview, error) {
	log := s.l.With(zap.String("operation", "UpdateReview"), zap.String("id", req.ID.String()))

	original, err := s.loadReview(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status != worker.ReviewStatusDraft {
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
	updated.OverallScore = worker.ComputeOverallScore(updated.Ratings)
	updated.Summary = strings.TrimSpace(req.Summary)
	updated.Strengths = strings.TrimSpace(req.Strengths)
	updated.Improvements = strings.TrimSpace(req.Improvements)
	updated.Goals = normalizeGoals(req.Goals)
	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.UpdateReview(ctx, &updated)
	if err != nil {
		log.Error("failed to update review", zap.Error(err))
		return nil, err
	}
	saved.Template = original.Template

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: saved.GetResourceID(),
		operation: permission.OpUpdate, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: "Review draft saved", log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeReview, permission.OpUpdate, saved.ID, req.UserID)
	return saved, nil
}

// mergeRatings keeps the template-derived shape and takes scores and
// comments from the request by key; unknown keys are ignored.
func mergeRatings(current, incoming []worker.ReviewRating) []worker.ReviewRating {
	byKey := make(map[string]worker.ReviewRating, len(incoming))
	for _, rating := range incoming {
		byKey[rating.Key] = rating
	}
	merged := make([]worker.ReviewRating, 0, len(current))
	for _, rating := range current {
		next := rating
		if in, ok := byKey[rating.Key]; ok {
			next.Score = in.Score
			next.Comment = strings.TrimSpace(in.Comment)
		}
		merged = append(merged, next)
	}
	return merged
}

func normalizeGoals(goals []worker.ReviewGoal) []worker.ReviewGoal {
	out := make([]worker.ReviewGoal, 0, len(goals))
	for _, goal := range goals {
		title := strings.TrimSpace(goal.Title)
		if title == "" {
			continue
		}
		if goal.ID == "" {
			goal.ID = pulid.MustNew("goal_").String()
		}
		if goal.Status == "" {
			goal.Status = worker.ReviewGoalStatusOpen
		}
		goal.Title = title
		out = append(out, goal)
	}
	return out
}

type ReviewStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

// SubmitReview finishes the draft and hands it to the worker to sign off;
// every item must be rated and a summary written.
func (s *Service) SubmitReview(
	ctx context.Context,
	req *ReviewStatusRequest,
) (*worker.PerformanceReview, error) {
	log := s.l.With(zap.String("operation", "SubmitReview"), zap.String("id", req.ID.String()))

	original, err := s.loadReview(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status != worker.ReviewStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only a draft can be submitted",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.ReviewStatusSubmitted
	updated.SubmittedAt = &now
	updated.OverallScore = worker.ComputeOverallScore(updated.Ratings)
	multiErr := errortypes.NewMultiError()
	updated.ValidateForSubmit(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.UpdateReview(ctx, &updated)
	if err != nil {
		log.Error("failed to submit review", zap.Error(err))
		return nil, err
	}
	saved.Template = original.Template

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: saved.GetResourceID(),
		operation: permission.OpSubmit, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: "Review submitted to the worker", log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeReview, permission.OpSubmit, saved.ID, req.UserID)
	s.notifyDriver(ctx, req.TenantInfo, saved, s.reviewerName(ctx, req.TenantInfo, saved.ReviewerID))
	return saved, nil
}

func (s *Service) reviewerName(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reviewerID pulid.ID,
) string {
	if reviewerID.IsNil() || s.userRepo == nil {
		return ""
	}
	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   tenantInfo,
		LookupUserID: reviewerID,
	})
	if err != nil || user == nil {
		return ""
	}
	return user.Name
}

// ReopenReview pulls a submitted review back to draft before the worker has
// signed it, so a mistake can be fixed.
func (s *Service) ReopenReview(
	ctx context.Context,
	req *ReviewStatusRequest,
) (*worker.PerformanceReview, error) {
	log := s.l.With(zap.String("operation", "ReopenReview"), zap.String("id", req.ID.String()))

	original, err := s.loadReview(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status != worker.ReviewStatusSubmitted {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only a submitted review the worker has not signed can be reopened",
		)
	}

	updated := *original
	updated.Status = worker.ReviewStatusDraft
	updated.SubmittedAt = nil
	saved, err := s.repo.UpdateReview(ctx, &updated)
	if err != nil {
		log.Error("failed to reopen review", zap.Error(err))
		return nil, err
	}
	saved.Template = original.Template

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: saved.GetResourceID(),
		operation: permission.OpReopen, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: "Review reopened as a draft", log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeReview, permission.OpReopen, saved.ID, req.UserID)
	return saved, nil
}

// AcknowledgeReview is the worker's sign-off from Dash.
func (s *Service) AcknowledgeReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	workerID pulid.ID,
	comment string,
) (*worker.PerformanceReview, error) {
	original, err := s.loadReview(ctx, tenantInfo, id, 0)
	if err != nil {
		return nil, err
	}
	if original.WorkerID != workerID {
		return nil, errortypes.NewNotFoundError("Review not found")
	}
	if original.Status != worker.ReviewStatusSubmitted {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"This review is not waiting for your sign-off",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.ReviewStatusAcknowledged
	updated.AcknowledgedAt = &now
	updated.WorkerComment = strings.TrimSpace(comment)
	saved, err := s.repo.UpdateReview(ctx, &updated)
	if err != nil {
		return nil, err
	}
	saved.Template = original.Template
	s.publish(ctx, tenantInfo, realtimeReview, permission.OpUpdate, saved.ID, pulid.Nil)
	return saved, nil
}

// CloseReview files the review and sets the next date from the template's
// cadence. A submitted review can be closed without the worker's sign-off;
// the absence is visible on the record.
func (s *Service) CloseReview(
	ctx context.Context,
	req *ReviewStatusRequest,
) (*worker.PerformanceReview, error) {
	log := s.l.With(zap.String("operation", "CloseReview"), zap.String("id", req.ID.String()))

	original, err := s.loadReview(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status != worker.ReviewStatusSubmitted && original.Status != worker.ReviewStatusAcknowledged {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Submit the review before closing it",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.ReviewStatusClosed
	updated.ClosedAt = &now
	updated.ClosedByID = req.UserID
	updated.NextReviewAt = updated.NextReviewDate(original.Template)
	saved, err := s.repo.UpdateReview(ctx, &updated)
	if err != nil {
		log.Error("failed to close review", zap.Error(err))
		return nil, err
	}
	saved.Template = original.Template

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: saved.GetResourceID(),
		operation: permission.OpClose, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: "Review closed", log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeReview, permission.OpClose, saved.ID, req.UserID)
	return saved, nil
}

// DeleteReview discards a draft that should never have been started.
func (s *Service) DeleteReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With(zap.String("operation", "DeleteReview"), zap.String("id", id.String()))

	original, err := s.loadReview(ctx, tenantInfo, id, 0)
	if err != nil {
		return err
	}
	if original.Status != worker.ReviewStatusDraft {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only drafts can be deleted",
		)
	}
	if err = s.repo.DeleteReview(ctx, &repositories.GetPerformanceReviewByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete review", zap.Error(err))
		return err
	}

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReview, resourceID: original.GetResourceID(),
		operation: permission.OpDelete, userID: userID, tenant: tenantInfo,
		current: original, comment: "Review draft deleted", log: log,
	})
	s.publish(ctx, tenantInfo, realtimeReview, permission.OpDelete, original.ID, userID)
	return nil
}

func (s *Service) loadReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	version int64,
) (*worker.PerformanceReview, error) {
	original, err := s.repo.GetReviewByID(ctx, &repositories.GetPerformanceReviewByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeTemplate: true,
	})
	if err != nil {
		return nil, err
	}
	if version > 0 && original.Version != version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Review was changed by someone else. Reload and try again",
		)
	}
	return original, nil
}
