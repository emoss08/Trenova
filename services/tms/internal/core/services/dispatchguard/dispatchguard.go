// Package dispatchguard holds the dispatch-hold gate shared by the driver and
// carrier assignment services.
package dispatchguard

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// EnsureNoDispatchHold blocks coverage changes while the shipment carries an
// active dispatch-blocking hold.
func EnsureNoDispatchHold(
	ctx context.Context,
	holdRepo repositories.ShipmentHoldRepository,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	hasHold, err := holdRepo.HasActiveDispatchHold(ctx, &repositories.ActiveShipmentHoldRequest{
		ShipmentID: shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if hasHold {
		return errortypes.NewBusinessError("Shipment has an active dispatch-blocking hold").
			WithParam("shipmentId", shipmentID.String())
	}

	return nil
}
