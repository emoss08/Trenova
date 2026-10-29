package invoiceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListInvoiceSharesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
}

type InvoiceShareRepository interface {
	Upsert(ctx context.Context, shares []*invoiceshare.InvoiceShare) error
	ListByInvoiceID(
		ctx context.Context,
		req *ListInvoiceSharesRequest,
	) ([]*invoiceshare.InvoiceShare, error)
}
