package driversettlementrecurring

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

// appendRecurringCycleLines adds everything tied to the pay-period cycle —
// recurring earnings, guaranteed minimum, carry-forward, recurring deductions,
// and advance recovery. Instant payouts skip all of it.
func (s *Cycle) AppendLines(
	ctx context.Context,
	req *CycleRequest,
	control *settlementcontrol.SettlementControl,
	settlement *driversettlement.Settlement,
	profile *driverpay.PayProfile,
	lines []*driversettlement.SettlementLine,
) ([]*driversettlement.SettlementLine, error) {
	lines, guaranteeExcluded, err := s.appendEarningLines(ctx, req, settlement, lines)
	if err != nil {
		return nil, err
	}

	var grossSoFar int64
	for _, line := range lines {
		if line.Category == driversettlement.LineCategoryEarning {
			grossSoFar += line.AmountMinor
		}
	}
	guaranteeBase := grossSoFar - guaranteeExcluded

	if profile != nil && profile.GuaranteedPeriodMinimumMinor > 0 &&
		guaranteeBase < profile.GuaranteedPeriodMinimumMinor {
		topUp := profile.GuaranteedPeriodMinimumMinor - guaranteeBase
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryGuaranteeTopUp,
			Description: "Guaranteed minimum pay top-up",
			AmountMinor: topUp,
		})
		settlement.AddException(
			driversettlement.ExceptionCodeGuaranteeApplied,
			driversettlement.ExceptionSeverityWarning,
			fmt.Sprintf("Guaranteed minimum applied: %s top-up",
				money.FormatMinor(topUp, settlement.CurrencyCode)),
		)
	}

	carryIn := s.resolveCarryForward(ctx, req)
	if carryIn < 0 {
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryCarryForward,
			Description: "Negative balance carried forward from prior settlement",
			AmountMinor: carryIn,
		})
	}

	deductionLines, capped, err := s.buildDeductionLines(ctx, req)
	if err != nil {
		return nil, err
	}
	lines = append(lines, deductionLines...)
	if capped {
		settlement.AddException(
			driversettlement.ExceptionCodeDeductionCapped,
			driversettlement.ExceptionSeverityWarning,
			"One or more deductions reached their cap this period",
		)
	}

	advanceLines, err := s.buildAdvanceLines(ctx, req, control, grossSoFar, lines)
	if err != nil {
		return nil, err
	}
	return append(lines, advanceLines...), nil
}

func (s *Cycle) resolveCarryForward(
	ctx context.Context,
	req *CycleRequest,
) int64 {
	latest, err := s.settlementRepo.GetLatestForWorker(
		ctx,
		driversettlementports.GetLatestSettlementForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil || latest == nil {
		return 0
	}
	return latest.CarryForwardOutMinor
}

func (s *Cycle) appendEarningLines(
	ctx context.Context,
	req *CycleRequest,
	settlement *driversettlement.Settlement,
	lines []*driversettlement.SettlementLine,
) ([]*driversettlement.SettlementLine, int64, error) {
	earningLines, capped, guaranteeExcluded, err := s.buildEarningLines(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	if capped {
		settlement.AddException(
			driversettlement.ExceptionCodeEarningCapped,
			driversettlement.ExceptionSeverityWarning,
			"One or more recurring earnings reached their cap this period",
		)
	}
	return append(lines, earningLines...), guaranteeExcluded, nil
}

func (s *Cycle) buildEarningLines(
	ctx context.Context,
	req *CycleRequest,
) (lines []*driversettlement.SettlementLine, capped bool, guaranteeExcluded int64, err error) {
	earnings, err := s.earningRepo.ListActiveForWorker(
		ctx,
		driverpayports.ListActiveEarningsForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			AsOf:       req.PeriodEnd,
		},
	)
	if err != nil {
		return nil, false, 0, err
	}

	lines = make([]*driversettlement.SettlementLine, 0, len(earnings))
	for _, earning := range earnings {
		if earning.Frequency == recurringpay.EarningFrequencyMonthly &&
			!s.monthlyRecurrenceApplies(ctx, req) {
			continue
		}
		amount := earning.NextAmountMinor()
		if amount <= 0 {
			continue
		}
		if remaining := earning.RemainingCapMinor(); remaining != nil &&
			amount == *remaining {
			capped = true
		}

		category := driversettlement.LineCategoryEarning
		if earning.PayCode != nil && earning.PayCode.LineIsReimbursement() {
			category = driversettlement.LineCategoryReimbursement
		}
		if category == driversettlement.LineCategoryEarning &&
			earning.PayCode != nil && !earning.PayCode.CountsTowardGuarantee {
			guaranteeExcluded += amount
		}

		earningID := earning.ID
		payCodeID := earning.PayCodeID
		lines = append(lines, &driversettlement.SettlementLine{
			Category:           category,
			Description:        earning.Description,
			AmountMinor:        amount,
			RecurringEarningID: &earningID,
			PayCodeID:          &payCodeID,
		})
	}
	return lines, capped, guaranteeExcluded, nil
}

func (s *Cycle) buildDeductionLines(
	ctx context.Context,
	req *CycleRequest,
) ([]*driversettlement.SettlementLine, bool, error) {
	deductions, err := s.deductionRepo.ListActiveForWorker(
		ctx,
		driverpayports.ListActiveDeductionsForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			AsOf:       req.PeriodEnd,
		},
	)
	if err != nil {
		return nil, false, err
	}

	lines := make([]*driversettlement.SettlementLine, 0, len(deductions))
	var capped bool
	for _, deduction := range deductions {
		if deduction.Frequency == recurringpay.DeductionFrequencyMonthly &&
			!s.monthlyRecurrenceApplies(ctx, req) {
			continue
		}
		amount := deduction.NextAmountMinor()
		if amount <= 0 {
			continue
		}
		if remaining := deduction.RemainingCapMinor(); remaining != nil &&
			amount == *remaining {
			capped = true
		}

		category := driversettlement.LineCategoryDeduction
		var escrowAccountID *pulid.ID
		if deduction.IsEscrowContribution() {
			category = driversettlement.LineCategoryEscrowContribution
			escrowAccountID = deduction.EscrowAccountID
			if account, escErr := s.escrowRepo.GetActiveForWorker(
				ctx,
				driverpayports.GetActiveEscrowAccountForWorkerRequest{
					TenantInfo: req.TenantInfo,
					WorkerID:   req.WorkerID,
				},
			); escErr == nil && account.TargetAmountMinor > 0 {
				remainingToTarget := account.TargetAmountMinor - account.BalanceMinor
				if remainingToTarget <= 0 {
					continue
				}
				amount = min(amount, remainingToTarget)
			}
		}

		deductionID := deduction.ID
		payCodeID := deduction.PayCodeID
		lines = append(lines, &driversettlement.SettlementLine{
			Category:             category,
			Description:          deduction.Description,
			AmountMinor:          -amount,
			RecurringDeductionID: &deductionID,
			EscrowAccountID:      escrowAccountID,
			PayCodeID:            &payCodeID,
		})
	}
	return lines, capped, nil
}

func (s *Cycle) monthlyRecurrenceApplies(
	ctx context.Context,
	req *CycleRequest,
) bool {
	latest, err := s.settlementRepo.GetLatestForWorker(
		ctx,
		driversettlementports.GetLatestSettlementForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil || latest == nil {
		return true
	}
	currentMonth := time.Unix(req.PeriodEnd, 0).UTC().Month()
	currentYear := time.Unix(req.PeriodEnd, 0).UTC().Year()
	latestMonth := time.Unix(latest.PeriodEnd, 0).UTC().Month()
	latestYear := time.Unix(latest.PeriodEnd, 0).UTC().Year()
	return currentMonth != latestMonth || currentYear != latestYear
}

func (s *Cycle) buildAdvanceLines(
	ctx context.Context,
	req *CycleRequest,
	control *settlementcontrol.SettlementControl,
	gross int64,
	existingLines []*driversettlement.SettlementLine,
) ([]*driversettlement.SettlementLine, error) {
	advances, err := s.advanceRepo.ListOutstandingForWorker(
		ctx,
		driverpayports.ListOutstandingAdvancesForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(advances) == 0 {
		return nil, nil
	}

	availableNet := gross
	for _, line := range existingLines {
		switch line.Category {
		case driversettlement.LineCategoryDeduction,
			driversettlement.LineCategoryEscrowContribution,
			driversettlement.LineCategoryCarryForward:
			availableNet += line.AmountMinor
		case driversettlement.LineCategoryGuaranteeTopUp,
			driversettlement.LineCategoryReimbursement,
			driversettlement.LineCategoryAdjustment:
			availableNet += line.AmountMinor
		case driversettlement.LineCategoryEarning,
			driversettlement.LineCategoryAdvanceRecovery:
		}
	}

	lines := make([]*driversettlement.SettlementLine, 0, len(advances))
	for _, advance := range advances {
		outstanding := advance.OutstandingMinor()
		if outstanding <= 0 {
			continue
		}
		recovery := outstanding
		if !control.AllowNegativeNet {
			if availableNet <= 0 {
				break
			}
			recovery = min(recovery, availableNet)
		}
		availableNet -= recovery

		advanceID := advance.ID
		description := "Advance recovery"
		if advance.Reference != "" {
			description += " (" + advance.Reference + ")"
		}
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryAdvanceRecovery,
			Description: description,
			AmountMinor: -recovery,
			AdvanceID:   &advanceID,
		})
	}
	return lines, nil
}
