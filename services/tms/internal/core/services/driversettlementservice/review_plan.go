package driversettlementservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type (
	DisputeChange = serviceports.RecordChange[driversettlement.Dispute]
	ExpenseChange = serviceports.RecordChange[driverpay.Expense]
)

type ResolveDisputePlan struct {
	Dispute          DisputeChange
	AdjustmentTarget *driversettlement.Settlement
}

type ReviewExpensePlan struct {
	ExpenseChange

	ReimbursementTarget *driversettlement.Settlement
}

func (s *Service) PlanStartDisputeReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	disputeID pulid.ID,
	actor *serviceports.RequestActor,
) (*DisputeChange, error) {
	if err := requireActor(actor, "Dispute review"); err != nil {
		return nil, err
	}
	original, err := s.disputeRepo.GetByID(ctx, repositories.GetSettlementDisputeByIDRequest{
		ID:         disputeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status != driversettlement.DisputeStatusOpen {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only open disputes can be moved to review",
		)
	}

	dispute := *original
	dispute.Status = driversettlement.DisputeStatusInReview
	return &DisputeChange{Before: original, After: &dispute}, nil
}

func (s *Service) PlanResolveDispute(
	ctx context.Context,
	req *ResolveDisputeRequest,
	actor *serviceports.RequestActor,
) (*ResolveDisputePlan, error) {
	if err := requireActor(actor, "Dispute resolution"); err != nil {
		return nil, err
	}
	if req.ResolutionNote == "" {
		return nil, errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"A resolution note is required",
		)
	}
	if !req.Approve && req.Adjustment != nil {
		return nil, errortypes.NewValidationError(
			"adjustment",
			errortypes.ErrInvalid,
			"A denied dispute cannot include an adjustment",
		)
	}

	original, err := s.disputeRepo.GetByID(ctx, repositories.GetSettlementDisputeByIDRequest{
		ID:         req.DisputeID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"This dispute has already been resolved",
		)
	}

	dispute := *original
	now := timeutils.NowUnix()
	if req.Approve {
		dispute.Status = driversettlement.DisputeStatusResolved
	} else {
		dispute.Status = driversettlement.DisputeStatusDenied
	}
	dispute.ResolutionNote = req.ResolutionNote
	dispute.ResolvedByID = &actor.UserID
	dispute.ResolvedAt = &now

	plan := &ResolveDisputePlan{Dispute: DisputeChange{Before: original, After: &dispute}}
	if req.Adjustment != nil {
		if plan.AdjustmentTarget, err = s.openDraftFor(
			ctx,
			req.TenantInfo,
			original.WorkerID,
		); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func (s *Service) openDraftFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*driversettlement.Settlement, error) {
	target, err := s.settlementRepo.GetOpenDraftForWorker(
		ctx,
		repositories.GetOpenDraftForWorkerRequest{TenantInfo: tenantInfo, WorkerID: workerID},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // no open draft: an off-cycle one is generated
		}
		return nil, err
	}
	return target, nil
}

func (s *Service) PlanReviewExpense(
	ctx context.Context,
	req *ReviewExpenseRequest,
	actor *serviceports.RequestActor,
) (*ReviewExpensePlan, error) {
	if err := requireActor(actor, "Expense review"); err != nil {
		return nil, err
	}
	if !req.Approve && strings.TrimSpace(req.Note) == "" {
		return nil, errortypes.NewValidationError(
			"note",
			errortypes.ErrRequired,
			"A note is required when rejecting an expense so the driver knows why",
		)
	}

	original, err := s.expenseRepo.GetByID(ctx, repositories.GetDriverExpenseByIDRequest{
		ID:         req.ExpenseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status != driverpay.ExpenseStatusPending {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only pending expenses can be reviewed",
		)
	}

	if req.Approve && original.ReceiptDocumentID == nil {
		control, controlErr := s.dashControlRepo.GetOrCreate(ctx, req.TenantInfo)
		if controlErr != nil {
			return nil, controlErr
		}
		if control.RequireExpenseReceipt {
			return nil, errortypes.NewValidationError(
				"receiptDocumentId",
				errortypes.ErrRequired,
				"This organization requires a receipt before an expense can be approved",
			)
		}
	}

	expense := *original
	now := timeutils.NowUnix()
	expense.ReviewNote = strings.TrimSpace(req.Note)
	expense.ReviewedByID = &actor.UserID
	expense.ReviewedAt = &now
	plan := &ReviewExpensePlan{ExpenseChange: ExpenseChange{Before: original, After: &expense}}
	if req.Approve {
		expense.Status = driverpay.ExpenseStatusReimbursed
		if plan.ReimbursementTarget, err = s.openDraftFor(
			ctx,
			req.TenantInfo,
			original.WorkerID,
		); err != nil {
			return nil, err
		}
	} else {
		expense.Status = driverpay.ExpenseStatusRejected
	}
	return plan, nil
}
