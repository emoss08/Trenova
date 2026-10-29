package shipmentbillingtransfer

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// transferGate is what a transfer checks before it reads a shipment's
// readiness: the shipment is outside the queue and ready to invoice, or
// completed and the caller asked for it to be marked ready first.
func transferGate(
	entity *shipment.Shipment,
	markCompleted bool,
) (markReady bool, code shipmentbillingports.BillingTransferFailureCode, err error) {
	if !entity.BillingTransferStatus.IsOutsideBillingQueue() {
		return false, shipmentbillingports.BillingTransferFailureAlreadyTransferred,
			errortypes.NewValidationError(
				"billingTransferStatus",
				errortypes.ErrInvalidOperation,
				"Shipment has already been transferred to billing",
			)
	}

	markReady = markCompleted && entity.Status == shipment.StatusCompleted
	if entity.Status != shipment.StatusReadyToInvoice && !markReady {
		return false, shipmentbillingports.BillingTransferFailureInvalidStatus, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalidOperation,
			"Shipment must be in ReadyToInvoice status to transfer to billing",
		)
	}

	return markReady, "", nil
}

func decisionFor(
	id pulid.ID,
	entity *shipment.Shipment,
) shipmentbillingports.BillingTransferDecision {
	decision := shipmentbillingports.BillingTransferDecision{
		ShipmentID:          id,
		MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{},
		ValidationFailures:  []shipmentbillingports.ShipmentBillingValidation{},
		Warnings:            []shipmentbillingports.ShipmentBillingWarning{},
	}
	if entity == nil {
		return decision
	}

	decision.ProNumber = entity.ProNumber
	decision.Status = entity.Status
	decision.BillingTransferStatus = entity.BillingTransferStatus
	decision.CustomerID = entity.CustomerID
	decision.TotalCharge = entity.TotalChargeAmount
	decision.DeliveredAt = entity.ActualDeliveryDate
	decision.Version = entity.Version
	if entity.Customer != nil {
		decision.CustomerName = entity.Customer.Name
	}

	return decision
}

func refuse(
	decision *shipmentbillingports.BillingTransferDecision,
	code shipmentbillingports.BillingTransferFailureCode,
	err error,
) {
	decision.Outcome = shipmentbillingports.BillingTransferOutcomeRefused
	if code == shipmentbillingports.BillingTransferFailureReturnToOperations {
		decision.Outcome = shipmentbillingports.BillingTransferOutcomeReturnToOperations
	}
	decision.FailureCode = code
	decision.Reason = err.Error()
}

func decide(
	decision *shipmentbillingports.BillingTransferDecision,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	markReady bool,
) {
	decision.MissingRequirements = readiness.MissingRequirements
	decision.ValidationFailures = readiness.ValidationFailures
	decision.Warnings = readiness.Warnings
	decision.PayerCount = len(readiness.Payers)
	decision.AutoApprove = readiness.ShouldAutoApproveBilling ||
		len(autoApprovePayerIDs(readiness)) > 0
	if decision.CustomerName == "" {
		decision.CustomerName = primaryPayer(readiness)
	}

	if code, err := billingTransferPolicyViolation(readiness); err != nil {
		refuse(decision, code, err)
		decision.AutoApprove = false

		return
	}

	decision.Outcome = shipmentbillingports.BillingTransferOutcomeTransfer
	if markReady {
		decision.Outcome = shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer
	}
}

func primaryPayer(readiness *shipmentbillingports.ShipmentBillingReadiness) string {
	for _, payer := range readiness.Payers {
		if payer.IsPrimary {
			return payer.PayerName
		}
	}

	return ""
}

func summarize(
	decisions []shipmentbillingports.BillingTransferDecision,
) *shipmentbillingports.BillingTransferPlan {
	plan := &shipmentbillingports.BillingTransferPlan{Decisions: decisions}
	for idx := range decisions {
		switch outcome := decisions[idx].Outcome; {
		case outcome.Transfers():
			plan.Transfer++
		case outcome == shipmentbillingports.BillingTransferOutcomeReturnToOperations:
			plan.Returned++
		default:
			plan.Refused++
		}
	}

	return plan
}
