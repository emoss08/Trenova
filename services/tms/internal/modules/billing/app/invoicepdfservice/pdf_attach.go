package invoicepdfservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// AttachGeneratedPDF records the document a background render produced as the
// invoice's PDF and tells anyone with the invoice open that it is ready.
func (s *Service) AttachGeneratedPDF(
	ctx context.Context,
	req *invoiceports.AttachGeneratedPDFRequest,
) (*invoice.Invoice, error) {
	current, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *current
	current.PDFDocumentID = req.DocumentID
	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.rec.LogAction(
		updated,
		req.Actor,
		permission.OpUpdate,
		&previous,
		updated,
		"Invoice PDF generated",
	)
	s.rec.PublishInvoiceChange(ctx, updated, req.Actor, invoicerealtime.RealtimeActionPDFGenerated)

	return updated, nil
}
