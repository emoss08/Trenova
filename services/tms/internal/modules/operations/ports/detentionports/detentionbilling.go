package detentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncharge"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// DetentionBillingHoldsRequest names the shipments an approval or an invoice is
// about to bill.
type DetentionBillingHoldsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
}

// DetentionBillingEvent is what happened to the invoices a billing sync reads.
type DetentionBillingEvent string

const DetentionBillingInvoiceCreated = DetentionBillingEvent("InvoiceCreated")

const DetentionBillingInvoiceVoided = DetentionBillingEvent("InvoiceVoided")

const DetentionBillingInvoiceAdjusted = DetentionBillingEvent("InvoiceAdjusted")

// SyncDetentionInvoiceBillingRequest names the invoices that were just created,
// voided or adjusted, and who did it.
type SyncDetentionInvoiceBillingRequest struct {
	TenantInfo    pagination.TenantInfo
	InvoiceIDs    []pulid.ID
	InvoiceNumber string
	Event         DetentionBillingEvent
	ActorUserID   pulid.ID
	Waivers       []DetentionWaiver
}

func (r *SyncDetentionInvoiceBillingRequest) WaiverFor(
	occurrenceID pulid.ID,
) (DetentionWaiver, bool) {
	for _, waiver := range r.Waivers {
		if waiver.OccurrenceID == occurrenceID {
			return waiver, true
		}
	}

	return DetentionWaiver{}, false
}

// SettleDetentionChargesRequest names a shipment about to be billed, loaded
// with its charges and allocations. Persist saves the settled charges; without
// it the shipment is only brought in line in memory, for a preview.
type SettleDetentionChargesRequest struct {
	TenantInfo  pagination.TenantInfo
	Shipment    *shipment.Shipment
	ActorUserID pulid.ID
	Persist     bool
}

// DetentionChargeSettler brings a shipment's detention charges in line with
// the decisions on its occurrences.
type DetentionChargeSettler interface {
	SettleShipmentCharges(
		ctx context.Context,
		req *SettleDetentionChargesRequest,
	) ([]detentioncharge.Change, error)
}

// DetentionBillingService keeps detention charges and invoices in step: a
// charge waiting on an approver stays off every invoice, a charge an invoice
// carries is marked billed with it, and one no standing invoice carries any
// more goes back to approved.
type DetentionBillingService interface {
	// HoldsForShipments returns the occurrences holding the shipments off an
	// invoice.
	HoldsForShipments(
		ctx context.Context,
		req *DetentionBillingHoldsRequest,
	) ([]*detention.DetentionOccurrence, error)
	// GuardShipments refuses, with the held occurrences named, when any of the
	// shipments has a charge waiting on an approver.
	GuardShipments(ctx context.Context, req *DetentionBillingHoldsRequest) error
	// SyncInvoiceBilling marks billed every billable occurrence whose charge
	// the standing invoices still carry, and releases every billed one whose
	// charge they no longer do. It runs in the caller's transaction.
	SyncInvoiceBilling(ctx context.Context, req *SyncDetentionInvoiceBillingRequest) error
	// SettleCharges brings the shipment's detention charges in line with the
	// waivers, disputes and approvals on its occurrences before it is billed.
	SettleCharges(ctx context.Context, req *SettleDetentionChargesRequest) error
}
