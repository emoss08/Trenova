package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func tenantOf(orgID, buID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}

func PlanIssueAdvance(entity *payadvance.PayAdvance, userID pulid.ID) error {
	entity.Status = payadvance.AdvanceStatusOutstanding
	entity.RecoveredMinor = 0
	entity.WrittenOffMinor = 0

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	entity.CreatedByID = userID
	return nil
}

func PlanWriteOffAdvance(
	entity *payadvance.PayAdvance,
	reason string,
	userID pulid.ID,
	now int64,
) error {
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A write-off reason is required",
		)
	}
	outstanding := entity.OutstandingMinor()
	if outstanding <= 0 {
		return errortypes.NewValidationError(
			"advanceId",
			errortypes.ErrInvalidOperation,
			"Advance has no outstanding balance to write off",
		)
	}
	entity.WrittenOffMinor += outstanding
	entity.WriteOffReason = reason
	entity.WrittenOffByID = userID
	entity.WrittenOffAt = &now
	entity.SyncStatus()
	return nil
}

func (s *Service) PlanOpenEscrowAccount(
	ctx context.Context,
	entity *payescrow.EscrowAccount,
	now int64,
) error {
	entity.Status = payescrow.EscrowAccountStatusActive
	entity.BalanceMinor = 0
	if entity.OpenedDate == 0 {
		entity.OpenedDate = now
	}
	tenantInfo := tenantOf(entity.OrganizationID, entity.BusinessUnitID)
	if entity.AnnualInterestRate.IsZero() {
		control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
		if err == nil {
			entity.AnnualInterestRate = control.DefaultEscrowInterestRate
		}
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	existing, err := s.escrowRepo.GetActiveForWorker(
		ctx,
		driverpayports.GetActiveEscrowAccountForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   entity.WorkerID,
		},
	)
	if err == nil && existing != nil {
		return errortypes.NewValidationError(
			"workerId",
			errortypes.ErrDuplicate,
			"Worker already has an active escrow account",
		)
	}
	return nil
}

func (s *Service) PlanUpdateEscrowAccount(
	ctx context.Context,
	entity *payescrow.EscrowAccount,
) (*payescrow.EscrowAccount, error) {
	previous, err := s.escrowRepo.GetByID(ctx, driverpayports.GetEscrowAccountByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
	})
	if err != nil {
		return nil, err
	}
	entity.BalanceMinor = previous.BalanceMinor
	entity.Status = previous.Status
	entity.OpenedDate = previous.OpenedDate
	entity.CurrencyCode = previous.CurrencyCode
	entity.ClosedDate = previous.ClosedDate
	entity.LastInterestAccrualDate = previous.LastInterestAccrualDate

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return previous, multiErr
	}
	return previous, nil
}

func PlanEscrowAdjustment(
	account *payescrow.EscrowAccount,
	req *EscrowAdjustmentRequest,
	userID pulid.ID,
) (*payescrow.EscrowTransaction, error) {
	if err := CheckEscrowAdjustment(req); err != nil {
		return nil, err
	}
	if account.Status != payescrow.EscrowAccountStatusActive {
		return nil, errortypes.NewValidationError(
			"accountId",
			errortypes.ErrInvalidOperation,
			"Escrow account is not active",
		)
	}
	if account.BalanceMinor+req.AmountMinor < 0 {
		return nil, errortypes.NewValidationError(
			"amountMinor",
			errortypes.ErrInvalid,
			"Adjustment would drive the escrow balance negative",
		)
	}
	return &payescrow.EscrowTransaction{
		Type:         payescrow.EscrowTransactionTypeAdjustment,
		AmountMinor:  req.AmountMinor,
		OccurredDate: req.OccurredDate,
		Description:  req.Description,
		CreatedByID:  userID,
	}, nil
}

func CheckEscrowAdjustment(req *EscrowAdjustmentRequest) error {
	if req.AmountMinor == 0 {
		return errortypes.NewValidationError(
			"amountMinor",
			errortypes.ErrInvalid,
			"Adjustment amount cannot be zero",
		)
	}
	if req.Description == "" {
		return errortypes.NewValidationError(
			"description",
			errortypes.ErrRequired,
			"Adjustment description is required",
		)
	}
	return nil
}

func PlanCloseEscrowAccount(
	account *payescrow.EscrowAccount,
	userID pulid.ID,
	now int64,
) (*payescrow.EscrowTransaction, error) {
	if account.Status != payescrow.EscrowAccountStatusActive {
		return nil, errortypes.NewValidationError(
			"accountId",
			errortypes.ErrInvalidOperation,
			"Escrow account is already closed",
		)
	}
	var refund *payescrow.EscrowTransaction
	if account.BalanceMinor > 0 {
		refund = &payescrow.EscrowTransaction{
			Type:         payescrow.EscrowTransactionTypeRefund,
			AmountMinor:  -account.BalanceMinor,
			OccurredDate: now,
			Description:  "Escrow balance refunded on account closure",
			CreatedByID:  userID,
		}
	}
	return refund, nil
}

func StampEscrowTransaction(account *payescrow.EscrowAccount, tx *payescrow.EscrowTransaction) {
	tx.OrganizationID = account.OrganizationID
	tx.BusinessUnitID = account.BusinessUnitID
	tx.EscrowAccountID = account.ID
	tx.BalanceAfterMinor = account.BalanceMinor + tx.AmountMinor
}

func CloseEscrowAccount(account *payescrow.EscrowAccount, now int64) {
	account.Status = payescrow.EscrowAccountStatusClosed
	account.ClosedDate = &now
}

type AssignmentPlan struct {
	Profile *driverpay.PayProfile
	Ended   []*payassignment.WorkerPayAssignment
}

func (s *Service) PlanAssignment(
	ctx context.Context,
	entity *payassignment.WorkerPayAssignment,
) (*AssignmentPlan, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	profile, err := s.profileRepo.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
		ID:                entity.PayProfileID,
		TenantInfo:        tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		IncludeComponents: true,
	})
	if err != nil {
		return nil, err
	}
	if err = validateRateOverrides(entity, profile); err != nil {
		return nil, err
	}

	overlapping, err := s.assignmentRepo.ListOverlapping(ctx, entity)
	if err != nil {
		return nil, err
	}
	ended, err := PlanEndOverlapping(entity, overlapping)
	if err != nil {
		return nil, err
	}
	return &AssignmentPlan{Profile: profile, Ended: ended}, nil
}

func PlanEndOverlapping(
	entity *payassignment.WorkerPayAssignment,
	overlapping []*payassignment.WorkerPayAssignment,
) ([]*payassignment.WorkerPayAssignment, error) {
	ended := make([]*payassignment.WorkerPayAssignment, 0, len(overlapping))
	for _, existing := range overlapping {
		if existing == nil {
			continue
		}
		if existing.EffectiveTo != nil && *existing.EffectiveTo <= entity.EffectiveFrom {
			continue
		}
		if existing.EffectiveFrom >= entity.EffectiveFrom {
			return nil, errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalid,
				"Worker already has a pay assignment starting on or after this date",
			)
		}
		closed := *existing
		endDate := entity.EffectiveFrom
		closed.EffectiveTo = &endDate
		ended = append(ended, &closed)
	}
	return ended, nil
}

func (s *Service) GetAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
) (*payassignment.WorkerPayAssignment, error) {
	return s.assignmentRepo.GetByID(ctx, tenantInfo, assignmentID)
}

func PlanEndAssignment(entity *payassignment.WorkerPayAssignment, endDate int64) error {
	if endDate <= entity.EffectiveFrom {
		return errortypes.NewValidationError(
			"endDate",
			errortypes.ErrInvalid,
			"End date must be after the assignment's effective from date",
		)
	}
	entity.EffectiveTo = &endDate
	return nil
}

func (s *Service) CheckDeduction(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
	autoLinkEscrow bool,
) error {
	tenantInfo := tenantOf(entity.OrganizationID, entity.BusinessUnitID)
	if err := s.resolvePayCode(
		ctx,
		tenantInfo,
		entity.PayCodeID,
		paycode.PayCodeDirectionDeduction,
	); err != nil {
		return err
	}
	if autoLinkEscrow && !entity.IsEscrowContribution() {
		account, err := s.escrowRepo.GetActiveForWorker(
			ctx,
			driverpayports.GetActiveEscrowAccountForWorkerRequest{
				TenantInfo: tenantInfo,
				WorkerID:   entity.WorkerID,
			},
		)
		if err != nil {
			return errortypes.NewValidationError(
				"escrowAccountId",
				errortypes.ErrRequired,
				"Worker has no active escrow account; open one before adding an escrow contribution",
			)
		}
		entity.EscrowAccountID = &account.ID
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) CheckEarning(ctx context.Context, entity *recurringpay.RecurringEarning) error {
	if err := s.resolvePayCode(
		ctx,
		tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		entity.PayCodeID,
		paycode.PayCodeDirectionEarning,
	); err != nil {
		return err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
