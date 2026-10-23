package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
)

// invoiceReader reads the invoice whose correction lineage
// invoice_list_adjustments lists.
type invoiceReader interface {
	GetByID(
		ctx context.Context,
		req invoiceports.GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
}
