package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetInvoiceDisputeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetOpenInvoiceDisputeRequest struct {
	InvoiceID  pulid.ID              `json:"invoiceId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListInvoiceDisputesByInvoiceIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"-"`
}

type InvoiceDisputeRepository interface {
	Create(ctx context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error)
	Update(ctx context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error)
	GetByID(ctx context.Context, req GetInvoiceDisputeByIDRequest) (*invoice.InvoiceDispute, error)
	// GetOpenByInvoiceID returns the invoice's open case, or a not-found error
	// when it has none.
	GetOpenByInvoiceID(
		ctx context.Context,
		req GetOpenInvoiceDisputeRequest,
	) (*invoice.InvoiceDispute, error)
	// ListByInvoiceIDs returns every case on the invoices, newest first, keyed
	// by invoice.
	ListByInvoiceIDs(
		ctx context.Context,
		req *ListInvoiceDisputesByInvoiceIDsRequest,
	) (map[pulid.ID][]*invoice.InvoiceDispute, error)
}
