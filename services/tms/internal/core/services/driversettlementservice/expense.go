package driversettlementservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const driverExpenseResource = "driver_expense"

type SubmitExpenseRequest struct {
	TenantInfo        pagination.TenantInfo
	WorkerID          pulid.ID
	SubmittedByUserID pulid.ID
	ShipmentID        *pulid.ID
	PayCodeID         *pulid.ID
	AmountMinor       int64
	Description       string
	IncurredDate      int64
}

type ReviewExpenseRequest struct {
	TenantInfo pagination.TenantInfo
	ExpenseID  pulid.ID
	Approve    bool
	Note       string
}

func (s *Service) ListExpensesConnection(
	ctx context.Context,
	req *repositories.ListDriverExpenseConnectionRequest,
) (*pagination.CursorListResult[*driverpay.Expense], error) {
	return s.expenseRepo.ListConnection(ctx, req)
}

func (s *Service) GetExpense(
	ctx context.Context,
	req repositories.GetDriverExpenseByIDRequest,
) (*driverpay.Expense, error) {
	return s.expenseRepo.GetByID(ctx, req)
}

func (s *Service) ListExpensesForWorker(
	ctx context.Context,
	req *repositories.ListDriverExpensesForWorkerRequest,
) ([]*driverpay.Expense, error) {
	return s.expenseRepo.ListForWorker(ctx, req)
}

func (s *Service) CountPendingExpenses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return s.expenseRepo.CountPending(ctx, tenantInfo)
}

func (s *Service) SubmitExpense(
	ctx context.Context,
	req *SubmitExpenseRequest,
) (*driverpay.Expense, error) {
	if req.PayCodeID != nil && !req.PayCodeID.IsNil() {
		payCode, err := s.payCodeRepo.GetByID(ctx, repositories.GetPayCodeByIDRequest{
			ID:         *req.PayCodeID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if payCode.Direction != driverpay.PayCodeDirectionEarning {
			return nil, errortypes.NewValidationError(
				"payCodeId",
				errortypes.ErrInvalid,
				"Expense reimbursements must use an earning pay code",
			)
		}
	}

	incurred := req.IncurredDate
	if incurred == 0 {
		incurred = timeutils.NowUnix()
	}

	entity := &driverpay.Expense{
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		WorkerID:          req.WorkerID,
		ShipmentID:        req.ShipmentID,
		PayCodeID:         req.PayCodeID,
		Status:            driverpay.ExpenseStatusPending,
		AmountMinor:       req.AmountMinor,
		CurrencyCode:      "USD",
		Description:       strings.TrimSpace(req.Description),
		IncurredDate:      incurred,
		SubmittedByUserID: req.SubmittedByUserID,
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.expenseRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.publishRealtimeInvalidation(
		ctx,
		driverExpenseResource,
		permission.OpCreate,
		created.ID,
		created.OrganizationID,
		created.BusinessUnitID,
		req.SubmittedByUserID,
	)
	s.notifyExpenseSubmitted(ctx, created)
	return created, nil
}

func (s *Service) notifyExpenseSubmitted(ctx context.Context, expense *driverpay.Expense) {
	if s.notificationService == nil {
		return
	}
	buID := expense.BusinessUnitID
	entity := &notification.Notification{
		OrganizationID: expense.OrganizationID,
		BusinessUnitID: &buID,
		EventType:      "driver_expense_submitted",
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Driver expense submitted",
		Message:        "A driver submitted an expense for reimbursement review.",
		Data:           map[string]any{"link": "/payroll/expenses"},
		RelatedEntities: map[string]any{
			"expenseId": expense.ID.String(),
			"workerId":  expense.WorkerID.String(),
		},
		Source: "driver_portal",
	}
	if _, err := s.notificationService.Create(ctx, entity); err != nil {
		s.l.Warn("failed to notify expense submission")
	}
}

func (s *Service) CancelExpense(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expenseID pulid.ID,
	workerID pulid.ID,
) (*driverpay.Expense, error) {
	expense, err := s.expenseRepo.GetByID(ctx, repositories.GetDriverExpenseByIDRequest{
		ID:         expenseID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if expense.WorkerID != workerID {
		return nil, errortypes.NewNotFoundError("Expense not found")
	}
	if expense.Status != driverpay.ExpenseStatusPending {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only pending expenses can be cancelled",
		)
	}

	expense.Status = driverpay.ExpenseStatusCancelled
	updated, err := s.expenseRepo.Update(ctx, expense)
	if err != nil {
		return nil, err
	}
	s.publishRealtimeInvalidation(
		ctx,
		driverExpenseResource,
		permission.OpCancel,
		updated.ID,
		updated.OrganizationID,
		updated.BusinessUnitID,
		tenantInfo.UserID,
	)
	return updated, nil
}

func (s *Service) SetExpenseReceipt(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expenseID pulid.ID,
	workerID pulid.ID,
	documentID pulid.ID,
) (*driverpay.Expense, error) {
	expense, err := s.expenseRepo.GetByID(ctx, repositories.GetDriverExpenseByIDRequest{
		ID:         expenseID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if expense.WorkerID != workerID {
		return nil, errortypes.NewNotFoundError("Expense not found")
	}
	if expense.Status.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"This expense has already been finalized",
		)
	}

	expense.ReceiptDocumentID = &documentID
	return s.expenseRepo.Update(ctx, expense)
}

// ReviewExpense approves or rejects a driver expense. Approval immediately
// applies a reimbursement adjustment to the worker's open draft settlement
// (creating an off-cycle draft when none exists) and marks the expense
// Reimbursed with the resulting settlement line linked for audit.
func (s *Service) ReviewExpense(
	ctx context.Context,
	req *ReviewExpenseRequest,
	actor *serviceports.RequestActor,
) (*driverpay.Expense, error) {
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

	expense, err := s.expenseRepo.GetByID(ctx, repositories.GetDriverExpenseByIDRequest{
		ID:         req.ExpenseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if expense.Status != driverpay.ExpenseStatusPending {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only pending expenses can be reviewed",
		)
	}

	if req.Approve && expense.ReceiptDocumentID == nil {
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

	now := timeutils.NowUnix()
	expense.ReviewNote = strings.TrimSpace(req.Note)
	expense.ReviewedByID = &actor.UserID
	expense.ReviewedAt = &now

	if req.Approve {
		if err = s.applyExpenseReimbursement(ctx, req.TenantInfo, expense, actor); err != nil {
			return nil, err
		}
		expense.Status = driverpay.ExpenseStatusReimbursed
	} else {
		expense.Status = driverpay.ExpenseStatusRejected
	}

	updated, err := s.expenseRepo.Update(ctx, expense)
	if err != nil {
		return nil, err
	}

	s.publishRealtimeInvalidation(
		ctx,
		driverExpenseResource,
		permission.OpApprove,
		updated.ID,
		updated.OrganizationID,
		updated.BusinessUnitID,
		actor.UserID,
	)
	s.notifyExpenseReviewed(ctx, req.TenantInfo, updated, req.Approve)
	return updated, nil
}

func (s *Service) applyExpenseReimbursement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expense *driverpay.Expense,
	actor *serviceports.RequestActor,
) error {
	target, err := s.settlementRepo.GetOpenDraftForWorker(
		ctx,
		repositories.GetOpenDraftForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   expense.WorkerID,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return err
		}
		target, err = s.generateDisputeSettlement(ctx, tenantInfo, expense.WorkerID, actor)
		if err != nil {
			return err
		}
	}

	description := "Expense reimbursement: " + expense.Description
	updated, err := s.AddAdjustmentLine(ctx, tenantInfo, target.ID, &AdjustmentLineInput{
		Description: description,
		AmountMinor: expense.AmountMinor,
		PayCodeID:   expense.PayCodeID,
	}, actor)
	if err != nil {
		return err
	}
	for _, line := range updated.Lines {
		if line.Description == description && line.AmountMinor == expense.AmountMinor {
			lineID := line.ID
			expense.SettlementLineID = &lineID
			break
		}
	}
	return nil
}

func (s *Service) notifyExpenseReviewed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expense *driverpay.Expense,
	approved bool,
) {
	if s.driverNotify == nil {
		return
	}
	title := "Expense rejected"
	message := "Your expense was rejected: " + expense.ReviewNote
	if approved {
		title = "Expense approved"
		message = "Your expense was approved and will be reimbursed on your next settlement."
	}
	s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   expense.WorkerID,
		EventType:  "dash.expense_reviewed",
		Priority:   notification.PriorityHigh,
		Title:      title,
		Message:    message,
		Link:       "/dash/money",
		RelatedEntities: map[string]any{
			"expenseId": expense.ID.String(),
		},
	})
}
