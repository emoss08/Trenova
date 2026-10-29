package shipmenteffects

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func ValidateShipmentNotLockedForBilling(entity *shipment.Shipment) *errortypes.MultiError {
	if entity.BillingTransferStatus.IsOutsideBillingQueue() {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"billingTransferStatus",
		errortypes.ErrInvalidOperation,
		"Shipment cannot be updated while in the billing queue. Send it back to operations from the billing queue to make changes.",
	)

	return multiErr
}
