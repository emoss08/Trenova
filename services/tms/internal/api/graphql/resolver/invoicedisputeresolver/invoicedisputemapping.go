package invoicedisputeresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	fieldInvoiceID  = "invoiceId"
	fieldInvoiceIDs = "invoiceIds"
)

// disputedInvoiceIDs are the invoices whose dispute cases a read names: one
// by invoiceId or several by invoiceIds, never both, and at most a page of
// them, in the order named.
func disputedInvoiceIDs(invoiceID *string, invoiceIDs []string) ([]pulid.ID, error) {
	named := invoiceID != nil && *invoiceID != ""
	switch {
	case named && len(invoiceIDs) > 0:
		return nil, errortypes.NewValidationError(fieldInvoiceIDs, errortypes.ErrInvalid,
			"Name the invoice by invoiceId or the invoices by invoiceIds, not both")
	case named:
		id, err := base.RequiredID(fieldInvoiceID, *invoiceID)
		if err != nil {
			return nil, err
		}
		return []pulid.ID{id}, nil
	case len(invoiceIDs) == 0:
		return nil, errortypes.NewValidationError(fieldInvoiceIDs, errortypes.ErrRequired,
			"Name at least one invoice")
	case len(invoiceIDs) > pagination.MaxLimit:
		return nil, errortypes.NewValidationError(fieldInvoiceIDs, errortypes.ErrInvalid,
			"At most 100 invoices can be read at once")
	}

	ids := make([]pulid.ID, 0, len(invoiceIDs))
	for _, raw := range invoiceIDs {
		id, err := base.RequiredID(fieldInvoiceIDs, raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return sliceutils.Dedupe(ids), nil
}
