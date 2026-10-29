package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) PreviewDecision(
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentDecisionRequest,
) (*invoiceadjustmentports.InvoiceAdjustmentDecisionPreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	adjustment, err := s.engine.GetDetail(
		ctx,
		&invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest{
			AdjustmentID: req.AdjustmentID,
			TenantInfo:   req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !adjustment.Status.CanBeRejected() {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only an adjustment awaiting approval or whose execution failed can be approved or rejected; this one is {0}",
			string(adjustment.Status),
		)
	}

	sourceInvoice, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         adjustment.OriginalInvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	decision := &invoiceadjustmentports.InvoiceAdjustmentDecisionPreview{
		Adjustment: adjustment,
		Invoice:    sourceInvoice,
	}
	if !req.Approve {
		return decision, nil
	}

	computation, err := s.engine.ComputePreview(
		ctx,
		&invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      adjustment.OriginalInvoiceID,
			Kind:           adjustment.Kind,
			RebillStrategy: adjustment.RebillStrategy,
			Reason:         adjustment.Reason,
			IdempotencyKey: adjustment.IdempotencyKey,
			TenantInfo:     req.TenantInfo,
			Lines:          s.engine.RequestLinesFromAdjustment(adjustment),
		},
		adjustment.ID,
	)
	if err != nil {
		return nil, err
	}
	decision.Figures = computation.Preview

	return decision, nil
}
