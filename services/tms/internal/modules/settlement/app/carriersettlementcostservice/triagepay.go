package carriersettlementcostservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var errNoInvoiceReading = errortypes.NewBusinessError(
	"The invoice has not been read yet, so it cannot be queued for carrier pay",
)

func (s *Service) QueueCarrierInvoice(
	ctx context.Context,
	req *inboundmessageports.QueueCarrierInvoiceRequest,
) error {
	if req.DocumentAIExtractionID.IsNil() {
		return errNoInvoiceReading
	}

	extractionID := req.DocumentAIExtractionID
	_, err := s.CreateMatch(ctx, &CreateMatchRequest{
		TenantInfo:             req.TenantInfo,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              req.CarrierID,
		InvoiceNumber:          req.InvoiceNumber,
		InvoiceTotalMinor:      req.TotalMinor,
		ShipmentID:             req.ShipmentID,
	}, req.Actor)

	return err
}

var _ inboundmessageports.TriageCarrierPayQueuer = (*Service)(nil)
