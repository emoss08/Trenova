package invoiceadjustmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func (s *Service) validateAttachments(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	preview *servicesports.InvoiceAdjustmentPreview,
) {
	fieldName := attachmentFieldName(req.AdjustmentID)
	if len(req.AttachmentIDs) == 0 {
		return
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        req.AttachmentIDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		appendPreviewError(preview, fieldName, "Failed to validate supporting documents")
		return
	}
	if len(docs) != len(req.AttachmentIDs) {
		appendPreviewError(preview, fieldName, "One or more supporting documents are invalid")
		return
	}
	for _, doc := range docs {
		if doc == nil || doc.Status == document.StatusArchived {
			appendPreviewError(
				preview,
				fieldName,
				"Archived supporting documents cannot be used for adjustments",
			)
			return
		}
	}
}

func (s *Service) buildDraftLines(
	sourceInvoice *invoice.Invoice,
	adjustmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) []*invoiceadjustment.InvoiceAdjustmentLine {
	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(sourceInvoice.Lines))
	for _, line := range sourceInvoice.Lines {
		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OrganizationID:          tenantInfo.OrgID,
			BusinessUnitID:          tenantInfo.BuID,
			AdjustmentID:            adjustmentID,
			OriginalInvoiceID:       sourceInvoice.ID,
			OriginalLineID:          line.ID,
			LineNumber:              line.LineNumber,
			Description:             line.Description,
			CreditQuantity:          line.Quantity,
			CreditAmount:            line.Amount.Abs(),
			RemainingEligibleAmount: line.Amount.Abs(),
			RebillQuantity:          line.Quantity,
			RebillAmount:            line.Amount.Abs(),
			ReplacementPayload:      map[string]any{},
		})
	}
	return lines
}

func (s *Service) buildAdjustmentLines(
	adjustmentID pulid.ID,
	invoiceID pulid.ID,
	inputs []*servicesports.InvoiceAdjustmentLineInput,
	tenantInfo pagination.TenantInfo,
) []*invoiceadjustment.InvoiceAdjustmentLine {
	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(inputs))
	for idx, line := range inputs {
		if line == nil {
			continue
		}
		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OrganizationID:          tenantInfo.OrgID,
			BusinessUnitID:          tenantInfo.BuID,
			AdjustmentID:            adjustmentID,
			OriginalInvoiceID:       invoiceID,
			OriginalLineID:          line.OriginalLineID,
			LineNumber:              idx + 1,
			Description:             line.Description,
			CreditQuantity:          line.CreditQuantity,
			CreditAmount:            line.CreditAmount,
			RemainingEligibleAmount: line.CreditAmount,
			RebillQuantity:          line.RebillQuantity,
			RebillAmount:            line.RebillAmount,
			ReplacementPayload:      line.ReplacementPayload,
		})
	}
	return lines
}

func (s *Service) buildDocumentReferences(
	ctx context.Context,
	adjustmentID pulid.ID,
	documentIDs []pulid.ID,
	sourceInvoice *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	actor *servicesports.RequestActor,
	fieldName string,
) ([]*invoiceadjustment.InvoiceAdjustmentDocumentReference, error) {
	if len(documentIDs) == 0 {
		return nil, nil
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        documentIDs,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, errortypes.NewValidationError(
			fieldName,
			errortypes.ErrInvalidOperation,
			"One or more supporting documents are invalid",
		)
	}
	if len(docs) != len(documentIDs) {
		return nil, errortypes.NewValidationError(
			fieldName,
			errortypes.ErrInvalidOperation,
			"One or more supporting documents are invalid",
		)
	}

	allowedShipmentIDs := make(map[string]struct{})
	for _, legID := range sourceInvoice.LegShipmentIDs() {
		allowedShipmentIDs[legID.String()] = struct{}{}
	}

	now := timeutils.NowUnix()
	refs := make([]*invoiceadjustment.InvoiceAdjustmentDocumentReference, 0, len(docs))
	for _, doc := range docs {
		if doc == nil || doc.Status == document.StatusArchived {
			return nil, errortypes.NewValidationError(
				fieldName,
				errortypes.ErrInvalidOperation,
				"Archived supporting documents cannot be used for adjustments",
			)
		}
		if _, allowed := allowedShipmentIDs[doc.ResourceID]; !allowed ||
			!strings.EqualFold(doc.ResourceType, "shipment") {
			return nil, errortypes.NewValidationError(
				fieldName,
				errortypes.ErrInvalidOperation,
				"Supporting evidence must reference documents from the invoice's shipments",
			)
		}

		refs = append(refs, &invoiceadjustment.InvoiceAdjustmentDocumentReference{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			AdjustmentID:         adjustmentID,
			DocumentID:           doc.ID,
			SelectedByID:         actor.UserID,
			SelectedAt:           &now,
			SnapshotFileName:     doc.FileName,
			SnapshotOriginalName: doc.OriginalName,
			SnapshotFileType:     doc.FileType,
			SnapshotResourceType: doc.ResourceType,
			SnapshotResourceID:   doc.ResourceID,
		})
	}

	return refs, nil
}

func attachmentFieldName(adjustmentID pulid.ID) string {
	if adjustmentID.IsNotNil() {
		return "referencedDocumentIds"
	}
	return "attachmentIds"
}

func (s *Service) requestLinesFromAdjustment(
	entity *invoiceadjustment.InvoiceAdjustment,
) []*servicesports.InvoiceAdjustmentLineInput {
	lines := make([]*servicesports.InvoiceAdjustmentLineInput, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, &servicesports.InvoiceAdjustmentLineInput{
			OriginalLineID:     line.OriginalLineID,
			CreditQuantity:     line.CreditQuantity,
			CreditAmount:       line.CreditAmount,
			RebillQuantity:     line.RebillQuantity,
			RebillAmount:       line.RebillAmount,
			Description:        line.Description,
			ReplacementPayload: line.ReplacementPayload,
		})
	}
	return lines
}

func sumInvoiceLines(
	lines []*invoice.InvoiceLine,
	lineType invoice.InvoiceLineType,
) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if lineType == "" || line.Type == lineType {
			total = total.Add(line.Amount)
		}
	}
	return total
}
