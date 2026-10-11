package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/uptrace/bun"
)

const draftPreviewIdempotencyKey = "draft-preview"

func (s *Service) SaveDraft(
	ctx context.Context,
	req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := validateSaveDraftRequest(req); err != nil {
		return nil, err
	}

	var saved *invoiceadjustment.InvoiceAdjustment
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		adjustmentID := req.AdjustmentID
		if adjustmentID.IsNil() {
			created, createErr := s.CreateDraft(
				txCtx,
				&invoiceadjustmentports.CreateDraftInvoiceAdjustmentRequest{
					InvoiceID:  req.InvoiceID,
					TenantInfo: req.TenantInfo,
				},
				actor,
			)
			if createErr != nil {
				return createErr
			}
			adjustmentID = created.ID
		}

		var updateErr error
		saved, updateErr = s.UpdateDraft(
			txCtx,
			&invoiceadjustmentports.UpdateDraftInvoiceAdjustmentRequest{
				AdjustmentID:          adjustmentID,
				Kind:                  req.Kind,
				RebillStrategy:        req.RebillStrategy,
				Reason:                req.Reason,
				ReferencedDocumentIDs: req.ReferencedDocumentIDs,
				Lines:                 req.Lines,
				TenantInfo:            req.TenantInfo,
			},
			actor,
		)

		return updateErr
	})
	if err != nil {
		return nil, err
	}

	return saved, nil
}

func validateSaveDraftRequest(req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest) error {
	if req == nil {
		return errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	multiErr := errortypes.NewMultiError()
	if req.AdjustmentID.IsNil() && req.InvoiceID.IsNil() {
		multiErr.Add(
			"invoiceId",
			errortypes.ErrRequired,
			"Name the invoice for a new draft, or the draft to change",
		)
	}
	if !req.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Adjustment kind is invalid")
	}
	if req.RebillStrategy != "" && !req.RebillStrategy.IsValid() {
		multiErr.Add("rebillStrategy", errortypes.ErrInvalid, "Rebill strategy is invalid")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) PreviewSaveDraft(
	ctx context.Context,
	req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustmentports.InvoiceAdjustmentDraftPreview, error) {
	if err := validateSaveDraftRequest(req); err != nil {
		return nil, err
	}

	var before *invoiceadjustment.InvoiceAdjustment
	invoiceID := req.InvoiceID
	if req.AdjustmentID.IsNotNil() {
		existing, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
			ID:         req.AdjustmentID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if existing.Status != invoiceadjustment.StatusDraft {
			return nil, errortypes.NewValidationError(
				"adjustmentId",
				errortypes.ErrInvalidOperation,
				"Only draft adjustments may be updated",
			)
		}
		before = existing
		invoiceID = existing.OriginalInvoiceID
	}

	sourceInvoice, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         invoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if _, err = s.buildDocumentReferences(
		ctx,
		req.AdjustmentID,
		req.ReferencedDocumentIDs,
		sourceInvoice,
		req.TenantInfo,
		actor,
		"referencedDocumentIds",
	); err != nil {
		return nil, err
	}

	after := draftAfter(before, sourceInvoice, req)
	after.Lines = s.buildAdjustmentLines(after.ID, sourceInvoice.ID, req.Lines, req.TenantInfo)

	computation, err := s.computePreview(ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
		AdjustmentID:   req.AdjustmentID,
		InvoiceID:      sourceInvoice.ID,
		Kind:           req.Kind,
		RebillStrategy: after.RebillStrategy,
		Reason:         req.Reason,
		IdempotencyKey: draftPreviewIdempotencyKey,
		AttachmentIDs:  req.ReferencedDocumentIDs,
		Lines:          req.Lines,
		TenantInfo:     req.TenantInfo,
	}, req.AdjustmentID)
	if err != nil {
		return nil, err
	}

	return &invoiceadjustmentports.InvoiceAdjustmentDraftPreview{
		Before:  before,
		After:   after,
		Invoice: sourceInvoice,
		Figures: computation.preview,
	}, nil
}

func draftAfter(
	before *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
) *invoiceadjustment.InvoiceAdjustment {
	after := &invoiceadjustment.InvoiceAdjustment{
		OrganizationID:          req.TenantInfo.OrgID,
		BusinessUnitID:          req.TenantInfo.BuID,
		OriginalInvoiceID:       sourceInvoice.ID,
		Status:                  invoiceadjustment.StatusDraft,
		ApprovalStatus:          invoiceadjustment.ApprovalStatusNotRequired,
		ReplacementReviewStatus: invoiceadjustment.ReplacementReviewStatusNotRequired,
		RebillStrategy:          invoiceadjustment.RebillStrategyCloneExact,
	}
	if before != nil {
		copied := *before
		after = &copied
	}
	after.Kind = req.Kind
	after.RebillStrategy = req.RebillStrategy
	after.Reason = req.Reason

	return after
}

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

	adjustment, err := s.GetDetail(ctx, &invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: req.AdjustmentID,
		TenantInfo:   req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if adjustment.Status != invoiceadjustment.StatusPendingApproval {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only an adjustment pending approval can be approved or rejected; this one is {0}",
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

	computation, err := s.computePreview(ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      adjustment.OriginalInvoiceID,
		Kind:           adjustment.Kind,
		RebillStrategy: adjustment.RebillStrategy,
		Reason:         adjustment.Reason,
		IdempotencyKey: adjustment.IdempotencyKey,
		TenantInfo:     req.TenantInfo,
		Lines:          s.requestLinesFromAdjustment(adjustment),
	}, adjustment.ID)
	if err != nil {
		return nil, err
	}
	decision.Figures = computation.preview

	return decision, nil
}
