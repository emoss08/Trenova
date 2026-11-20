package invoicesendservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
)

const replacedInvoiceSendError = "A rebill replaced this invoice, and customers see only current invoices. Send the replacement invoice instead."

func (s *Service) hiddenAsReplaced(ctx context.Context, entity *invoice.Invoice) (bool, error) {
	if entity.SupersededByInvoiceID.IsNil() {
		return false, nil
	}
	control, err := s.adjustmentRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return false, err
	}
	return !control.ShowsReplacedInvoicesExternally(), nil
}
