package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// ratedBy is what priced the item's shipment: its rate agreement, read as the
// agreement's own row and nothing under it, or else the formula template the
// shipment already carries. An agreement that cannot be read falls back to the
// template rather than failing the item.
func (s *Service) ratedBy(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) *billingqueue.RatedBy {
	shp := item.Shipment
	if shp == nil {
		return nil
	}
	if shp.RateAgreementID != nil && shp.RateAgreementID.IsNotNil() && s.agreements != nil {
		agreement, err := s.agreements.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
			RateAgreementID: *shp.RateAgreementID,
			TenantInfo:      tenantInfo,
		})
		if err == nil && agreement != nil {
			return &billingqueue.RatedBy{
				Kind: billingqueue.RatedByAgreement,
				ID:   agreement.ID,
				Code: agreement.Code,
				Name: agreement.Name,
			}
		}
		if err != nil {
			s.l.Warn("could not read the rate agreement that priced a billing queue item",
				zap.String("billingQueueItemId", item.ID.String()),
				zap.String("rateAgreementId", shp.RateAgreementID.String()),
				zap.Error(err))
		}
	}
	if template := shp.FormulaTemplate; template != nil && template.ID.IsNotNil() {
		return &billingqueue.RatedBy{
			Kind: billingqueue.RatedByTemplate,
			ID:   template.ID,
			Name: template.Name,
		}
	}

	return nil
}
