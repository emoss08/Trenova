package bankreceiptservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// MatchPreview is a receipt and its open work item before and after a match
// to a payment, as Match would leave them.
type MatchPreview struct {
	ReceiptBefore  *bankreceipt.BankReceipt
	ReceiptAfter   *bankreceipt.BankReceipt
	Payment        *customerpayment.Payment
	WorkItemBefore *bankreceiptworkitem.WorkItem
	WorkItemAfter  *bankreceiptworkitem.WorkItem
}

// PreviewMatch is Match without the writes.
func (s *Service) PreviewMatch(
	ctx context.Context,
	req *serviceports.MatchBankReceiptRequest,
	actor *serviceports.RequestActor,
) (*MatchPreview, error) {
	return s.planStoredMatch(ctx, req, actor)
}

// PreviewMatchPaymentRequest is a match to a payment that is itself only
// previewed, so it is carried rather than read.
type PreviewMatchPaymentRequest struct {
	ReceiptID  pulid.ID
	TenantInfo pagination.TenantInfo
	Payment    *customerpayment.Payment
}

// PreviewMatchPayment previews matching a receipt to a payment a post would
// record in the same step.
func (s *Service) PreviewMatchPayment(
	ctx context.Context,
	req *PreviewMatchPaymentRequest,
	actor *serviceports.RequestActor,
) (*MatchPreview, error) {
	if req == nil || req.Payment == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if err := requireMatchActor(actor); err != nil {
		return nil, err
	}

	return s.planMatch(ctx, req.ReceiptID, req.TenantInfo, req.Payment, actor)
}

func (s *Service) planStoredMatch(
	ctx context.Context,
	req *serviceports.MatchBankReceiptRequest,
	actor *serviceports.RequestActor,
) (*MatchPreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if err := requireMatchActor(actor); err != nil {
		return nil, err
	}

	payment, err := s.paymentRepo.GetByID(
		ctx,
		repositoryports.GetCustomerPaymentByIDRequest{
			ID:         req.PaymentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return s.planMatch(ctx, req.ReceiptID, req.TenantInfo, payment, actor)
}

func requireMatchActor(actor *serviceports.RequestActor) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Bank receipt matching requires an authenticated user",
		)
	}

	return nil
}

// planMatch checks that the receipt and the payment can be matched and
// applies the match to a copy of each record it changes: the receipt, and
// the reconciliation work item the match closes.
func (s *Service) planMatch(
	ctx context.Context,
	receiptID pulid.ID,
	tenantInfo pagination.TenantInfo,
	payment *customerpayment.Payment,
	actor *serviceports.RequestActor,
) (*MatchPreview, error) {
	receipt, err := s.repo.GetByID(
		ctx,
		repositoryports.GetBankReceiptByIDRequest{ID: receiptID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if receipt.Status == bankreceipt.StatusMatched {
		return nil, errortypes.NewBusinessError("Bank receipt is already matched")
	}
	if payment.Status != customerpayment.StatusPosted {
		return nil, errortypes.NewBusinessError("Only posted customer payments can be matched")
	}
	if receipt.AmountMinor != payment.AmountMinor {
		return nil, errortypes.NewBusinessError(
			"Bank receipt amount must match customer payment amount",
		)
	}

	before := *receipt
	now := timeutils.NowUnix()
	receipt.Status = bankreceipt.StatusMatched
	receipt.MatchedCustomerPaymentID = payment.ID
	receipt.MatchedAt = &now
	receipt.MatchedByID = actor.UserID
	receipt.UpdatedByID = actor.UserID

	plan := &MatchPreview{ReceiptBefore: &before, ReceiptAfter: receipt, Payment: payment}
	if s.workItemRepo == nil {
		return plan, nil
	}

	item := s.activeWorkItem(ctx, tenantInfo, receipt.ID)
	if item == nil {
		return plan, nil
	}
	itemBefore := *item
	item.Status = bankreceiptworkitem.StatusResolved
	item.ResolutionType = bankreceiptworkitem.ResolutionMatchedToPayment
	item.ResolutionNote = "Resolved by matching bank receipt to customer payment"
	item.ResolvedByUserID = actor.UserID
	item.ResolvedAt = &now
	item.UpdatedByID = actor.UserID
	plan.WorkItemBefore = &itemBefore
	plan.WorkItemAfter = item

	return plan, nil
}
