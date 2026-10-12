package invoiceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// OpenInvoiceDisputeRequest raises a dispute case on a posted invoice.
type OpenInvoiceDisputeRequest struct {
	InvoiceID      pulid.ID                  `json:"invoiceId"`
	TenantInfo     pagination.TenantInfo     `json:"-"`
	ReasonCode     invoice.DisputeReasonCode `json:"reasonCode"`
	DisputedAmount decimal.Decimal           `json:"disputedAmount"`
	Notes          string                    `json:"notes"`
}

// ResolveInvoiceDisputeRequest closes an open case with an outcome. A credit
// or write-off must name the executed adjustment that settled it.
type ResolveInvoiceDisputeRequest struct {
	DisputeID              pulid.ID                  `json:"disputeId"`
	TenantInfo             pagination.TenantInfo     `json:"-"`
	Resolution             invoice.DisputeResolution `json:"resolution"`
	ResolutionAdjustmentID pulid.ID                  `json:"resolutionAdjustmentId"`
	ResolutionNotes        string                    `json:"resolutionNotes"`
}

// WithdrawInvoiceDisputeRequest drops an open case the customer no longer
// pursues, without recording an outcome.
type WithdrawInvoiceDisputeRequest struct {
	DisputeID  pulid.ID              `json:"disputeId"`
	TenantInfo pagination.TenantInfo `json:"-"`
	Notes      string                `json:"notes"`
}

type InvoiceDisputePreview struct {
	Before        *invoice.InvoiceDispute
	After         *invoice.InvoiceDispute
	InvoiceBefore *invoice.Invoice
	InvoiceAfter  *invoice.Invoice
}

type InvoiceDisputeService interface {
	Open(
		ctx context.Context,
		req *OpenInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	Resolve(
		ctx context.Context,
		req *ResolveInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	Withdraw(
		ctx context.Context,
		req *WithdrawInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	PreviewOpen(
		ctx context.Context,
		req *OpenInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceDisputePreview, error)
	PreviewResolve(
		ctx context.Context,
		req *ResolveInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceDisputePreview, error)
	PreviewWithdraw(
		ctx context.Context,
		req *WithdrawInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceDisputePreview, error)
}
