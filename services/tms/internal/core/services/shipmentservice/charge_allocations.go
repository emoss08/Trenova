package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/allocationcheck"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// validateChargeAllocations checks a save's allocations before anything is
// written. The rules live in allocationcheck so the billing queue enforces the
// same ones when a charge is reassigned.
func (s *service) validateChargeAllocations(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	return allocationcheck.Validate(ctx, allocationcheck.Deps{
		CustomerRepo:         s.customerRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		Logger:               s.l,
	}, entity, "chargeAllocations")
}

func previewForResolution(entity *shipment.Shipment) *shipment.Shipment {
	return allocationcheck.PreviewForResolution(entity)
}
