package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MyReviews lists reviews the driver can see: submitted ones waiting for
// sign-off and everything after. Drafts stay with the reviewer.
func (s *Service) MyReviews(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.PerformanceReview, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.reviews.ListReviews(ctx, tenantInfo, wrk.ID, []worker.ReviewStatus{
		worker.ReviewStatusSubmitted,
		worker.ReviewStatusAcknowledged,
		worker.ReviewStatusClosed,
	})
}

// AcknowledgeMyReview is the driver's sign-off, with an optional comment the
// reviewer sees.
func (s *Service) AcknowledgeMyReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reviewID pulid.ID,
	comment string,
) (*worker.PerformanceReview, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	saved, err := s.reviews.AcknowledgeReview(ctx, tenantInfo, reviewID, wrk.ID, comment)
	if err != nil {
		return nil, err
	}
	s.notifyDispatch(
		ctx,
		tenantInfo,
		"review_acknowledged",
		"Review signed off",
		wrk.FirstName+" "+wrk.LastName+" signed off on \""+saved.Title+"\". Close it when you are ready.",
		"/hr/workers?tab=reviews",
		map[string]any{"workerId": wrk.ID.String(), "reviewId": saved.ID.String()},
	)
	return saved, nil
}
