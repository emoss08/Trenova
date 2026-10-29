package shipmentbillingtransfer

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/shared/pulid"
)

const unexpectedBillingTransferMessage = "The shipment could not be transferred to billing because of an unexpected error"

type billingTransferAttemptParams struct {
	ShipmentID                  pulid.ID
	BillType                    billingqueue.BillType
	MarkCompletedReadyToInvoice bool
	Actor                       *permissionports.RequestActor
	SuppressExceptionNotices    bool
	// Cache holds the answers that do not vary by shipment, so a run of five
	// thousand does not ask the same organization-level question five thousand
	// times. It lives for exactly one bulk call.
	Cache *shipmentbillingreadiness.BillingReadinessCache
}

type billingTransferAttempt struct {
	entity      *shipment.Shipment
	readiness   *shipmentbillingports.ShipmentBillingReadiness
	item        *billingqueue.BillingQueueItem
	items       []*billingqueue.BillingQueueItem
	markedReady bool
	failure     shipmentbillingports.BillingTransferFailureCode
	err         error
}

func (a *billingTransferAttempt) fail(
	code shipmentbillingports.BillingTransferFailureCode,
	err error,
) *billingTransferAttempt {
	a.failure = code
	a.err = err
	return a
}

// autoApprovePayerIDs lists the payers whose own profile clears clean freight
// through the queue, so a split shipment can auto-approve one payer's item while
// another payer's waits for a biller.
func autoApprovePayerIDs(readiness *shipmentbillingports.ShipmentBillingReadiness) []pulid.ID {
	if readiness == nil {
		return nil
	}
	ids := make([]pulid.ID, 0, len(readiness.Payers))
	for _, payer := range readiness.Payers {
		if payer.ShouldAutoApproveBilling {
			ids = append(ids, payer.PayerID)
		}
	}

	return ids
}

func newBulkTransferResult(
	shipmentID pulid.ID,
	attempt *billingTransferAttempt,
) shipmentbillingports.BulkTransferToBillingResult {
	result := shipmentbillingports.BulkTransferToBillingResult{
		ShipmentID:           shipmentID,
		MarkedReadyToInvoice: attempt.markedReady,
		Item:                 attempt.item,
		Items:                attempt.items,
		MissingRequirements:  []shipmentbillingports.ShipmentBillingRequirement{},
		ValidationFailures:   []shipmentbillingports.ShipmentBillingValidation{},
	}
	if attempt.entity != nil {
		result.ProNumber = attempt.entity.ProNumber
	}
	if attempt.readiness != nil {
		result.MissingRequirements = attempt.readiness.MissingRequirements
		result.ValidationFailures = attempt.readiness.ValidationFailures
	}

	if attempt.err == nil {
		result.Success = true
		return result
	}

	result.FailureCode = attempt.failure
	result.Err = attempt.err
	result.Error = unexpectedBillingTransferMessage
	if attempt.failure != shipmentbillingports.BillingTransferFailureUnexpected {
		result.Error = attempt.err.Error()
	}

	return result
}
