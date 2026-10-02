package shipmentstate

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func ValidateCancel(entity *shipment.Shipment) error {
	if entity == nil {
		return errortypes.NewNotFoundError("Shipment not found")
	}

	if entity.IsCanceled() {
		return errortypes.NewBusinessError("shipment is already canceled")
	}

	if entity.StatusEquals(shipment.StatusInvoiced) {
		return cancelConflict(errortypes.NewConflictError(
			"An invoiced shipment cannot be canceled. Credit or void its invoice first.",
		))
	}

	if !entity.BillingTransferStatus.IsOutsideBillingQueue() {
		return cancelConflict(errortypes.NewConflictError(
			"A shipment in the billing queue cannot be canceled. Send it back to operations from the billing queue first.",
		))
	}

	if !CanTransitionShipmentStatus(entity.Status, shipment.StatusCanceled) {
		return cancelConflict(errortypes.NewConflictError(
			"A shipment with status {0} cannot be canceled.",
			entity.Status,
		))
	}

	return nil
}

func cancelConflict(err *errortypes.ConflictError) *errortypes.ConflictError {
	err.Code = errortypes.ErrInvalidOperation
	return err
}
