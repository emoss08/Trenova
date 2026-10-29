package invoicevalidation

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// refuseVoidedInvoice keeps a voided document out of the customer's hands: it
// is still readable inside Trenova, but it is never rendered or delivered.
func RefuseVoided(entity *invoice.Invoice, action string) error {
	if entity == nil || entity.Status != invoice.StatusVoided {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"Invoice {0} has been voided and cannot be {1}",
		entity.Number,
		action,
	)
}
