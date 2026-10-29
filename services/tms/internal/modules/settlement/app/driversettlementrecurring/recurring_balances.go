package driversettlementrecurring

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Cycle) Apply(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *permissionports.RequestActor,
) error {
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		amount := -line.AmountMinor
		switch line.Category {
		case driversettlement.LineCategoryDeduction:
			if err := s.applyRecurringDeduction(ctx, entity, line, amount); err != nil {
				return err
			}
		case driversettlement.LineCategoryEscrowContribution:
			if err := s.applyEscrowContribution(ctx, entity, line, amount,
				actor); err != nil {
				return err
			}
		case driversettlement.LineCategoryAdvanceRecovery:
			if err := s.applyAdvanceRecovery(ctx, entity, line, amount); err != nil {
				return err
			}
		case driversettlement.LineCategoryEarning,
			driversettlement.LineCategoryReimbursement:
			if err := s.applyRecurringEarning(ctx, entity, line); err != nil {
				return err
			}
		case driversettlement.LineCategoryGuaranteeTopUp,
			driversettlement.LineCategoryCarryForward,
			driversettlement.LineCategoryAdjustment:
		}
	}
	return nil
}

func (s *Cycle) applyRecurringEarning(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
) error {
	if line.RecurringEarningID == nil || line.RecurringEarningID.IsNil() {
		return nil
	}
	earning, err := s.earningRepo.GetByID(
		ctx,
		driverpayports.GetRecurringEarningByIDRequest{
			ID: *line.RecurringEarningID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}
	earning.PaidToDateMinor += line.AmountMinor
	if remaining := earning.RemainingCapMinor(); remaining != nil && *remaining == 0 {
		earning.Status = recurringpay.EarningStatusCompleted
	}
	_, err = s.earningRepo.Update(ctx, earning)
	return err
}

func (s *Cycle) applyRecurringDeduction(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.RecurringDeductionID == nil || line.RecurringDeductionID.IsNil() {
		return nil
	}
	deduction, err := s.deductionRepo.GetByID(
		ctx,
		driverpayports.GetRecurringDeductionByIDRequest{
			ID: *line.RecurringDeductionID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}
	deduction.DeductedToDateMinor += amount
	if remaining := deduction.RemainingCapMinor(); remaining != nil && *remaining == 0 {
		deduction.Status = recurringpay.DeductionStatusCompleted
	}
	_, err = s.deductionRepo.Update(ctx, deduction)
	return err
}

func (s *Cycle) applyEscrowContribution(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
	actor *permissionports.RequestActor,
) error {
	if line.EscrowAccountID == nil || line.EscrowAccountID.IsNil() {
		return nil
	}
	if err := s.applyRecurringDeduction(ctx, entity, line, amount); err != nil {
		return err
	}
	return s.appendEscrowTransaction(ctx, entity, *line.EscrowAccountID,
		payescrow.EscrowTransactionTypeContribution, amount,
		"Escrow contribution from settlement "+entity.SettlementNumber, actor)
}

func (s *Cycle) applyAdvanceRecovery(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.AdvanceID == nil || line.AdvanceID.IsNil() {
		return nil
	}
	advance, err := s.advanceRepo.GetByID(ctx, driverpayports.GetPayAdvanceByIDRequest{
		ID: *line.AdvanceID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}
	advance.RecoveredMinor += amount
	advance.SyncStatus()
	_, err = s.advanceRepo.Update(ctx, advance)
	return err
}

func (s *Cycle) Reverse(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *permissionports.RequestActor,
) error {
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		amount := -line.AmountMinor
		switch line.Category {
		case driversettlement.LineCategoryDeduction:
			if err := s.reverseRecurringDeduction(ctx, entity, line, amount); err != nil {
				return err
			}
		case driversettlement.LineCategoryEscrowContribution:
			if err := s.reverseRecurringDeduction(ctx, entity, line, amount); err != nil {
				return err
			}
			if line.EscrowAccountID != nil && !line.EscrowAccountID.IsNil() {
				if err := s.appendEscrowTransaction(ctx, entity, *line.EscrowAccountID,
					payescrow.EscrowTransactionTypeAdjustment, -amount,
					"Escrow contribution reversed; settlement "+
						entity.SettlementNumber+" voided", actor); err != nil {
					return err
				}
			}
		case driversettlement.LineCategoryAdvanceRecovery:
			if err := s.reverseAdvanceRecovery(ctx, entity, line, amount); err != nil {
				return err
			}
		case driversettlement.LineCategoryEarning,
			driversettlement.LineCategoryReimbursement:
			if err := s.reverseRecurringEarning(ctx, entity, line); err != nil {
				return err
			}
		case driversettlement.LineCategoryGuaranteeTopUp,
			driversettlement.LineCategoryCarryForward,
			driversettlement.LineCategoryAdjustment:
		}
	}
	return nil
}

func (s *Cycle) reverseRecurringEarning(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
) error {
	if line.RecurringEarningID == nil || line.RecurringEarningID.IsNil() {
		return nil
	}
	earning, err := s.earningRepo.GetByID(
		ctx,
		driverpayports.GetRecurringEarningByIDRequest{
			ID: *line.RecurringEarningID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}
	earning.PaidToDateMinor = max(earning.PaidToDateMinor-line.AmountMinor, 0)
	if earning.Status == recurringpay.EarningStatusCompleted {
		if remaining := earning.RemainingCapMinor(); remaining == nil || *remaining > 0 {
			earning.Status = recurringpay.EarningStatusActive
		}
	}
	_, err = s.earningRepo.Update(ctx, earning)
	return err
}

func (s *Cycle) reverseRecurringDeduction(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.RecurringDeductionID == nil || line.RecurringDeductionID.IsNil() {
		return nil
	}
	deduction, err := s.deductionRepo.GetByID(
		ctx,
		driverpayports.GetRecurringDeductionByIDRequest{
			ID: *line.RecurringDeductionID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}
	deduction.DeductedToDateMinor = max(deduction.DeductedToDateMinor-amount, 0)
	if deduction.Status == recurringpay.DeductionStatusCompleted {
		if remaining := deduction.RemainingCapMinor(); remaining == nil || *remaining > 0 {
			deduction.Status = recurringpay.DeductionStatusActive
		}
	}
	_, err = s.deductionRepo.Update(ctx, deduction)
	return err
}

func (s *Cycle) reverseAdvanceRecovery(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.AdvanceID == nil || line.AdvanceID.IsNil() {
		return nil
	}
	advance, err := s.advanceRepo.GetByID(ctx, driverpayports.GetPayAdvanceByIDRequest{
		ID: *line.AdvanceID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}
	advance.RecoveredMinor = max(advance.RecoveredMinor-amount, 0)
	advance.SyncStatus()
	_, err = s.advanceRepo.Update(ctx, advance)
	return err
}

func (s *Cycle) appendEscrowTransaction(
	ctx context.Context,
	entity *driversettlement.Settlement,
	accountID pulid.ID,
	txType payescrow.EscrowTransactionType,
	amount int64,
	description string,
	actor *permissionports.RequestActor,
) error {
	account, err := s.escrowRepo.GetByID(ctx, driverpayports.GetEscrowAccountByIDRequest{
		ID: accountID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}
	settlementID := entity.ID
	tx := &payescrow.EscrowTransaction{
		OrganizationID:    account.OrganizationID,
		BusinessUnitID:    account.BusinessUnitID,
		EscrowAccountID:   account.ID,
		Type:              txType,
		AmountMinor:       amount,
		BalanceAfterMinor: account.BalanceMinor + amount,
		OccurredDate:      timeutils.NowUnix(),
		Description:       description,
		SettlementID:      &settlementID,
		CreatedByID:       actor.UserIDOrNil(),
	}
	multiErr := errortypes.NewMultiError()
	tx.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	if _, err = s.escrowRepo.AppendTransaction(ctx, tx); err != nil {
		return err
	}
	account.BalanceMinor = tx.BalanceAfterMinor
	_, err = s.escrowRepo.Update(ctx, account)
	return err
}
