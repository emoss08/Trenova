package billingqueuereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) ListActivity(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueActivityRequest,
) (*billingqueueports.BillingQueueEventPage, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing queue activity is unavailable")
	}
	if _, err := s.repo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.reviewRepo.ListEvents(ctx, &billingqueueports.ListBillingQueueEventsRequest{
		TenantInfo: req.TenantInfo,
		ItemID:     req.ItemID,
		Limit:      req.Limit,
		BeforeAt:   req.BeforeAt,
		BeforeID:   req.BeforeID,
	})
}

func (s *Service) Neighbors(
	ctx context.Context,
	req *billingqueueports.GetBillingQueueNeighborsRequest,
) (*billingqueueports.BillingQueueNeighbors, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("The billing queue's order is unavailable")
	}

	return s.reviewRepo.GetNeighbors(ctx, req)
}

// maxSummaries bounds a summaries read to a page of the queue table.
const maxSummaries = 200

func (s *Service) Summaries(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueSummariesRequest,
) ([]*billingqueueports.BillingQueueItemSummary, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing queue summaries are unavailable")
	}
	if len(req.ItemIDs) > maxSummaries {
		return nil, errortypes.NewValidationError(
			"ids", errortypes.ErrInvalid, "Ask for at most 200 items at a time",
		)
	}

	return s.reviewRepo.ListSummaries(ctx, req)
}
