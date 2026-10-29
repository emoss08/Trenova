package shipmentbillingtransfer

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
)

func readinessAllowsAutoTransfer(
	status shipmentvocab.Status,
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
	case shipmentvocab.StatusReadyToInvoice:
		return true
	case shipmentvocab.StatusCompleted:
		return readiness.ShouldAutoMarkReadyToInvoice
	default:
		return false
	}
}
