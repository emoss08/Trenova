package invoicevoidservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevoid"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	maxVoidReasonLength         = 1000
	draftReturnedToReviewReason = "Billing queue item returned to review to correct charges"
)

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
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.VoidInvoiceResult, error) {
	entity, err := s.planVoid(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if entity.Status == invoice.StatusDraft {
		return s.voidDraft(ctx, req, actor)
	}

	return s.voidPosted(ctx, req, entity, actor)
}

func validateVoidRequest(req *invoiceports.VoidInvoiceRequest) *errortypes.MultiError {
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
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.VoidInvoiceResult, error) {
	var result *invoiceports.VoidInvoiceResult

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, txErr := s.repo.LockForUpdate(txCtx, invoiceports.GetInvoiceByIDRequest{
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

		result, txErr = s.voidLockedDraft(txCtx, entity, req, actor, "Invoice voided")
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DiscardDraftForReview voids the draft an approved billing queue item produced
// when the item goes back to review. The draft was priced from the charges as
// they were at approval, so keeping it would let a re-approval hand back the
// stale document instead of one built from the corrected charges. A posted
// invoice is refused: it is on the ledger and only a void can take it back.
func (s *Service) DiscardDraftForReview(
	ctx context.Context,
	req *invoiceports.DiscardDraftForReviewRequest,
	actor *permissionports.RequestActor,
) error {
	return s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, err := s.repo.GetByBillingQueueItemID(
			txCtx,
			invoiceports.GetInvoiceByBillingQueueItemIDRequest{
				BillingQueueItemID: req.BillingQueueItemID,
				TenantInfo:         req.TenantInfo,
			},
		)
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		if err != nil {
			return err
		}

		entity, err := s.repo.LockForUpdate(txCtx, invoiceports.GetInvoiceByIDRequest{
			ID:         existing.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}
		switch entity.Status { //nolint:exhaustive // only a draft can be voided back to review
		case invoice.StatusVoided:
			return nil
		case invoice.StatusDraft:
		default:
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Invoice {0} is already {1}; void it to send the freight back to review",
				entity.Number,
				string(entity.Status),
			)
		}

		_, err = s.voidLockedDraft(
			txCtx,
			entity,
			&invoiceports.VoidInvoiceRequest{
				InvoiceID:   entity.ID,
				TenantInfo:  req.TenantInfo,
				Reason:      draftReturnedToReviewReason,
				Disposition: invoice.VoidDispositionRebill,
			},
			actor,
			"Draft invoice discarded; billing queue item returned to review",
		)
		return err
	})
}

func (s *Service) voidLockedDraft(
	ctx context.Context,
	entity *invoice.Invoice,
	req *invoiceports.VoidInvoiceRequest,
	actor *permissionports.RequestActor,
	message string,
) (*invoiceports.VoidInvoiceResult, error) {
	auditActor := actor.AuditActor()
	previous := *entity
	now := timeutils.NowUnix()
	if err := s.markDraftVoided(ctx, entity, req, actor, now); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	released, err := invoicevoid.Release(
		ctx,
		s.voidDeps(),
		invoicevoid.Params{
			Invoice:     updated,
			Disposition: req.Disposition,
			ActorUserID: actor.UserID,
			Reason:      entity.VoidReason,
			WasPosted:   false,
			Now:         now,
		},
	)
	if err != nil {
		return nil, err
	}

	s.rec.LogAction(updated, auditActor, permission.OpCancel, &previous, updated, message)
	s.rec.PublishInvalidation(ctx, updated, auditActor, "updated", updated)

	return &invoiceports.VoidInvoiceResult{
		Invoice:              updated,
		ReleasedQueueItemIDs: released,
	}, nil
}

func (s *Service) voidPosted(
	ctx context.Context,
	req *invoiceports.VoidInvoiceRequest,
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
) (*invoiceports.VoidInvoiceResult, error) {
	if err := s.refuseVoidingPosted(entity); err != nil {
		return nil, err
	}

	// The reason and disposition are recorded before the reversal runs, so an
	// approver sees them and the engine has them when it marks the invoice.
	// Both happen in one transaction: a reversal the engine refuses must not
	// leave a posted invoice carrying a void reason it never got.
	previous := *entity
	var (
		adjustment *invoiceadjustment.InvoiceAdjustment
		reloaded   *invoice.Invoice
	)
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity.VoidReason = strings.TrimSpace(req.Reason)
		entity.VoidDisposition = req.Disposition
		if _, txErr := s.repo.Update(txCtx, entity); txErr != nil {
			return txErr
		}

		var txErr error
		adjustment, txErr = s.adjustmentService.Submit(
			txCtx,
			voidReversalRequest(entity, req),
			actor,
		)
		if txErr != nil {
			return txErr
		}

		reloaded, txErr = s.repo.GetByID(txCtx, invoiceports.GetInvoiceByIDRequest{
			ID:         entity.ID,
			TenantInfo: req.TenantInfo,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	result := &invoiceports.VoidInvoiceResult{
		Invoice:         reloaded,
		AdjustmentID:    adjustment.ID,
		PendingApproval: adjustment.Status != invoiceadjustment.StatusExecuted,
	}
	auditActor := actor.AuditActor()
	comment := "Invoice void requested; awaiting reversal approval"
	if !result.PendingApproval {
		comment = "Invoice voided by full reversal"
	}
	s.rec.LogAction(reloaded, auditActor, permission.OpCancel, &previous, reloaded, comment)
	s.rec.PublishInvalidation(ctx, reloaded, auditActor, "updated", reloaded)

	return result, nil
}

func (s *Service) numberer() billingnumber.Numberer {
	return billingnumber.New(s.sequenceGenerator, s.sequenceProvider, s.customerRepo)
}

func (s *Service) voidDeps() invoicevoid.Deps {
	return invoicevoid.Deps{
		BillingQueueRepo:     s.billingQueueRepo,
		OrderRepo:            s.orderRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		ShipmentRepo:         s.shipmentRepo,
		InvoiceRepo:          s.repo,
		OrderDerivation:      s.orderDerivation,
		DetentionBilling:     s.detentionBilling,
		Renumber:             s.numberer().ForItem,
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
		&latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest{
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
	memos, err := s.repo.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
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
