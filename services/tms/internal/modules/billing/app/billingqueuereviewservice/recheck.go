package billingqueuereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) Recheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) (*billingqueueports.RecheckResult, error) {
	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                itemID,
		TenantInfo:            tenantInfo,
		ExpandShipmentDetails: true,
		Recheck:               true,
	})
	if err != nil {
		return nil, err
	}

	s.activity.PublishInvalidation(ctx, item, kernelports.AuditActor{}, "updated", item)

	return &billingqueueports.RecheckResult{
		ItemID: item.ID,
		Status: item.Status,
		Ready:  item.Review != nil && item.Review.Ready,
	}, nil
}
