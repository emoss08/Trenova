package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
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

const (
	DetentionBillingInvoiceCreated  = DetentionBillingEvent("InvoiceCreated")
	DetentionBillingInvoiceVoided   = DetentionBillingEvent("InvoiceVoided")
	DetentionBillingInvoiceAdjusted = DetentionBillingEvent("InvoiceAdjusted")
)

// SyncDetentionInvoiceBillingRequest names the invoices that were just created,
// voided or adjusted, and who did it.
type SyncDetentionInvoiceBillingRequest struct {
	TenantInfo    pagination.TenantInfo
	InvoiceIDs    []pulid.ID
	InvoiceNumber string
	Event         DetentionBillingEvent
	ActorUserID   pulid.ID
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
}
