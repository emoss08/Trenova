package invoicevoidservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) planVoid(
	ctx context.Context,
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if multiErr := validateVoidRequest(req); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = s.refuseVoidWithStandingLateCharges(ctx, entity, req.TenantInfo); err != nil {
		return nil, err
	}

	switch entity.Status {
	case invoice.StatusDraft, invoice.StatusPosted:
		return entity, nil
	case invoice.StatusVoided:
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice {0} is already voided",
			entity.Number,
		)
	default:
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice cannot be voided from its current status",
		)
	}
}

func refuseAppliedInvoice(entity *invoice.Invoice) error {
	if entity.AppliedAmountMinor <= 0 {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"Unapply the customer payments and credit memos on this invoice before voiding it",
	)
}

func (s *Service) markDraftVoided(
	ctx context.Context,
	entity *invoice.Invoice,
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
	now int64,
) error {
	if err := refuseAppliedInvoice(entity); err != nil {
		return err
	}

	entity.Status = invoice.StatusVoided
	entity.VoidedAt = &now
	entity.VoidedByID = actor.UserID
	entity.VoidReason = strings.TrimSpace(req.Reason)
	entity.VoidDisposition = req.Disposition
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return multiErr
	}

	return nil
}

func (s *Service) refuseVoidingPosted(entity *invoice.Invoice) error {
	if err := refuseAppliedInvoice(entity); err != nil {
		return err
	}
	if s.adjustmentService == nil {
		return errortypes.NewConflictError(
			"Invoice adjustments are unavailable; a posted invoice cannot be voided",
		)
	}

	return nil
}

func voidReversalRequest(
	entity *invoice.Invoice,
	req *invoiceports.VoidInvoiceRequest,
) *invoiceadjustmentports.InvoiceAdjustmentRequest {
	return &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      entity.ID,
		Kind:           invoiceadjustment.KindFullReversal,
		Reason:         strings.TrimSpace(req.Reason),
		IdempotencyKey: "invoice-void:" + entity.ID.String(),
		TenantInfo:     req.TenantInfo,
	}
}

func (s *Service) PreviewVoid(
	ctx context.Context,
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoiceVoidPreview, error) {
	entity, err := s.planVoid(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *entity
	preview := &invoiceports.InvoiceVoidPreview{Before: entity, After: &after}
	if entity.Status == invoice.StatusDraft {
		if err = s.markDraftVoided(ctx, &after, req, actor, timeutils.NowUnix()); err != nil {
			return nil, err
		}
		return preview, nil
	}

	if err = s.refuseVoidingPosted(entity); err != nil {
		return nil, err
	}
	after.VoidReason = strings.TrimSpace(req.Reason)
	after.VoidDisposition = req.Disposition
	preview.Posted = true
	if preview.Reversal, err = s.adjustmentService.Preview(
		ctx,
		voidReversalRequest(entity, req),
		actor,
	); err != nil {
		return nil, err
	}

	return preview, nil
}
