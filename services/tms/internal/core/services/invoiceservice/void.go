package invoiceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicevoid"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const maxVoidReasonLength = 1000

// VoidInvoice takes an invoice out of circulation.
//
// A draft never reached the ledger, so it is voided in place and its freight is
// released at once. A posted invoice is voided through a full-reversal
// adjustment: the credit memo reverses the receivable and the engine marks the
// invoice the moment the reversal executes, which may be after an approver has
// looked at it. Either way the invoice keeps its number, so a voided document
// stays readable in the audit trail.
func (s *Service) VoidInvoice(
	ctx context.Context,
	req *servicesports.VoidInvoiceRequest,
	actor *servicesports.RequestActor,
) (*servicesports.VoidInvoiceResult, error) {
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

	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
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
	case invoice.StatusDraft:
		return s.voidDraft(ctx, req, actor)
	case invoice.StatusPosted:
		return s.voidPosted(ctx, req, entity, actor)
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

func validateVoidRequest(req *servicesports.VoidInvoiceRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	reason := strings.TrimSpace(req.Reason)
	switch {
	case reason == "":
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the invoice is being voided")
	case len(reason) > maxVoidReasonLength:
		multiErr.Add(
			"reason",
			errortypes.ErrInvalid,
			"Reason must be at most {0} characters",
			maxVoidReasonLength,
		)
	}
	if !req.Disposition.IsValid() {
		multiErr.Add(
			"disposition",
			errortypes.ErrInvalid,
			"Disposition must be Rebill or DoNotRebill",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) voidDraft(
	ctx context.Context,
	req *servicesports.VoidInvoiceRequest,
	actor *servicesports.RequestActor,
) (*servicesports.VoidInvoiceResult, error) {
	result := &servicesports.VoidInvoiceResult{}
	auditActor := actor.AuditActor()

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.repo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if entity.Status != invoice.StatusDraft {
			return errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Invoice is no longer a draft; reload and try again",
			)
		}
		if entity.AppliedAmountMinor > 0 {
			return errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Unapply the customer payments and credit memos on this invoice before voiding it",
			)
		}

		previous := *entity
		now := timeutils.NowUnix()
		entity.Status = invoice.StatusVoided
		entity.VoidedAt = &now
		entity.VoidedByID = actor.UserID
		entity.VoidReason = strings.TrimSpace(req.Reason)
		entity.VoidDisposition = req.Disposition
		if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
			return multiErr
		}
		updated, txErr := s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		released, txErr := invoicevoid.Release(
			txCtx,
			s.voidDeps(req.TenantInfo),
			invoicevoid.Params{
				Invoice:     updated,
				Disposition: req.Disposition,
				ActorUserID: actor.UserID,
				Reason:      entity.VoidReason,
				WasPosted:   false,
				Now:         now,
			},
		)
		if txErr != nil {
			return txErr
		}

		result.Invoice = updated
		result.ReleasedQueueItemIDs = released
		s.logAction(updated, auditActor, permission.OpCancel, &previous, updated, "Invoice voided")
		s.publishInvalidation(txCtx, updated, auditActor, "updated", updated)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) voidPosted(
	ctx context.Context,
	req *servicesports.VoidInvoiceRequest,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) (*servicesports.VoidInvoiceResult, error) {
	if entity.AppliedAmountMinor > 0 {
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Unapply the customer payments and credit memos on this invoice before voiding it",
		)
	}
	if entity.BillType == billingqueue.BillTypeCreditMemo && entity.AppliedAmountMinor > 0 {
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Unapply this credit memo from the invoices it settles before voiding it",
		)
	}
	if s.adjustmentService == nil {
		return nil, errortypes.NewConflictError(
			"Invoice adjustments are unavailable; a posted invoice cannot be voided",
		)
	}

	// The reason and disposition are recorded before the reversal runs, so an
	// approver sees them and the engine has them when it marks the invoice.
	previous := *entity
	entity.VoidReason = strings.TrimSpace(req.Reason)
	entity.VoidDisposition = req.Disposition
	if _, err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}

	adjustment, err := s.adjustmentService.Submit(ctx, &servicesports.InvoiceAdjustmentRequest{
		InvoiceID:      entity.ID,
		Kind:           invoiceadjustment.KindFullReversal,
		Reason:         entity.VoidReason,
		IdempotencyKey: "invoice-void:" + entity.ID.String(),
		TenantInfo:     req.TenantInfo,
	}, actor)
	if err != nil {
		return nil, err
	}

	reloaded, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	result := &servicesports.VoidInvoiceResult{
		Invoice:         reloaded,
		AdjustmentID:    adjustment.ID,
		PendingApproval: adjustment.Status != invoiceadjustment.StatusExecuted,
	}
	auditActor := actor.AuditActor()
	comment := "Invoice void requested; awaiting reversal approval"
	if !result.PendingApproval {
		comment = "Invoice voided by full reversal"
	}
	s.logAction(reloaded, auditActor, permission.OpCancel, &previous, reloaded, comment)
	s.publishInvalidation(ctx, reloaded, auditActor, "updated", reloaded)

	return result, nil
}

func (s *Service) voidDeps(tenantInfo pagination.TenantInfo) invoicevoid.Deps {
	return invoicevoid.Deps{
		BillingQueueRepo:     s.billingQueueRepo,
		OrderRepo:            s.orderRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		ShipmentRepo:         s.shipmentRepo,
		InvoiceRepo:          s.repo,
		OrderDerivation:      s.orderDerivation,
		DetentionBilling:     s.detentionBilling,
		Renumber: func(ctx context.Context, billType billingqueue.BillType) (string, error) {
			return s.renumberBillingItem(ctx, tenantInfo, billType)
		},
	}
}

// renumberBillingItem mints a fresh billing number for an item released to be
// billed again, because the voided invoice keeps the one it had.
func (s *Service) renumberBillingItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	billType billingqueue.BillType,
) (string, error) {
	switch billType {
	case billingqueue.BillTypeCreditMemo:
		return s.sequenceGenerator.GenerateCreditMemoNumber(
			ctx,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			"",
			"",
		)
	case billingqueue.BillTypeDebitMemo:
		return s.sequenceGenerator.GenerateDebitMemoNumber(
			ctx,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			"",
			"",
		)
	default:
		return s.sequenceGenerator.GenerateInvoiceNumber(
			ctx,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			"",
			"",
		)
	}
}

// refuseVoidWithStandingLateCharges keeps an invoice on the books while a
// late-charge memo raised on it still stands: void the memos first, so the
// customer is never charged interest on a debt that no longer exists.
func (s *Service) refuseVoidWithStandingLateCharges(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) error {
	if s.lateChargeRepo == nil || entity == nil {
		return nil
	}
	grouped, err := s.lateChargeRepo.ListBySourceInvoiceIDs(
		ctx,
		&repositories.ListLateChargeAssessmentsByInvoiceIDsRequest{
			TenantInfo: tenantInfo,
			InvoiceIDs: []pulid.ID{entity.ID},
		},
	)
	if err != nil {
		return err
	}
	assessments := grouped[entity.ID]
	if len(assessments) == 0 {
		return nil
	}

	memoIDs := make([]pulid.ID, 0, len(assessments))
	seen := make(map[pulid.ID]struct{}, len(assessments))
	for _, assessment := range assessments {
		if assessment == nil || assessment.DebitMemoInvoiceID.IsNil() {
			continue
		}
		if _, dup := seen[assessment.DebitMemoInvoiceID]; dup {
			continue
		}
		seen[assessment.DebitMemoInvoiceID] = struct{}{}
		memoIDs = append(memoIDs, assessment.DebitMemoInvoiceID)
	}
	if len(memoIDs) == 0 {
		return nil
	}
	memos, err := s.repo.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
		TenantInfo: tenantInfo,
		InvoiceIDs: memoIDs,
	})
	if err != nil {
		return err
	}
	standing := make([]string, 0, len(memos))
	for _, memo := range memos {
		if memo != nil && memo.Status != invoice.StatusVoided {
			standing = append(standing, memo.Number)
		}
	}
	if len(standing) == 0 {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"Void the late-charge memo(s) {0} raised on this invoice first",
		strings.Join(standing, ", "),
	)
}
