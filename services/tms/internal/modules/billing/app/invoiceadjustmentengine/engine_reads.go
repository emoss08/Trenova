package invoiceadjustmentengine

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Engine) GetDetail(
	ctx context.Context,
	req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	adjustmentDocs, err := s.documentRepo.GetByResourceID(
		ctx,
		&documentports.GetDocumentsByResourceRequest{
			TenantInfo:          req.TenantInfo,
			ResourceID:          req.AdjustmentID.String(),
			ResourceType:        adjustmentDocumentResourceType,
			IncludeDocumentType: true,
		},
	)
	if err == nil {
		entity.AdjustmentDocuments = adjustmentDocs
	}

	return entity, nil
}

func (s *Engine) Preview(
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	_ *permissionports.RequestActor,
) (*invoiceadjustmentports.InvoiceAdjustmentPreview, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	computation, err := s.ComputePreview(ctx, req, pulid.ID(""))
	if err != nil {
		return nil, err
	}

	return computation.Preview, nil
}
