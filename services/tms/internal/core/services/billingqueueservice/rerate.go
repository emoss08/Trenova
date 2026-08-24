package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// rerateShipment reprices a shipment whose rate a biller has just changed.
//
// Changing the rating method or the base rate here is the same act as changing
// it on the shipment itself: the contract no longer describes what is being
// billed. So the shipment stops counting as auto-rated, and the difference
// between what the contract charged and what the invoice will say is recorded
// where the rate leakage report reads it.
func (s *service) rerateShipment(
	ctx context.Context,
	shp *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	departed := shp.AutoRated
	if departed {
		shp.ClearAutoRating()
	}

	control, _ := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})

	if err := s.commercial.Recalculate(ctx, shp, control, userID); err != nil {
		return err
	}

	return s.commercial.RecordRateDeparture(ctx, shp, userID, departed)
}
