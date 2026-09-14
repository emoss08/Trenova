package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListInvoiceSharesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
}

type InvoiceShareRepository interface {
	Upsert(ctx context.Context, shares []*invoice.InvoiceShare) error
	ListByInvoiceID(
		ctx context.Context,
		req *ListInvoiceSharesRequest,
	) ([]*invoice.InvoiceShare, error)
}
