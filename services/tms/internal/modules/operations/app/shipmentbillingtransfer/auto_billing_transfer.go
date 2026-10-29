package shipmentbillingtransfer

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
)

func readinessAllowsAutoTransfer(
	status shipment.Status,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) bool {
	if readiness == nil ||
		readiness.Policy.BillingQueueTransferMode != billingcontrol.BillingQueueTransferModeAutomaticWhenReady {
		return false
	}
	if _, violation := billingTransferPolicyViolation(readiness); violation != nil {
		return false
	}

	switch status { //nolint:exhaustive // only these two may be transferred
	case shipment.StatusReadyToInvoice:
		return true
	case shipment.StatusCompleted:
		return readiness.ShouldAutoMarkReadyToInvoice
	default:
		return false
	}
}
