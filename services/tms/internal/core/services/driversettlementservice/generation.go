package driversettlementservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type GenerateBatchRequest struct {
	TenantInfo  pagination.TenantInfo
	PeriodStart int64
	PeriodEnd   int64
	Name        string
	Notes       string
}

type PeriodBounds struct {
	PeriodStart int64 `json:"periodStart"`
	PeriodEnd   int64 `json:"periodEnd"`
	PayDate     int64 `json:"payDate"`
}

func ResolveCurrentPeriod(control *tenant.SettlementControl, now int64) PeriodBounds {
	nowTime := time.Unix(now, 0).UTC()
	endDay := time.Weekday(control.PeriodEndDayOfWeek)

	daysBack := int(nowTime.Weekday() - endDay)
	if daysBack < 0 {
		daysBack += 7
	}
	periodEndDate := time.Date(
		nowTime.Year(), nowTime.Month(), nowTime.Day(),
		0, 0, 0, 0, time.UTC,
	).AddDate(0, 0, -daysBack+1)

	var periodStartDate time.Time
	switch control.PayPeriodFrequency {
	case tenant.PayPeriodFrequencyWeekly:
		periodStartDate = periodEndDate.AddDate(0, 0, -7)
	case tenant.PayPeriodFrequencyBiweekly:
		periodStartDate = periodEndDate.AddDate(0, 0, -14)
	case tenant.PayPeriodFrequencyMonthly:
		periodStartDate = periodEndDate.AddDate(0, -1, 0)
	default:
		periodStartDate = periodEndDate.AddDate(0, 0, -7)
	}

	periodEnd := periodEndDate.Unix()
	return PeriodBounds{
		PeriodStart: periodStartDate.Unix(),
		PeriodEnd:   periodEnd,
		PayDate:     periodEndDate.AddDate(0, 0, control.PayDelayDays).Unix(),
	}
}

func (s *Service) GenerateBatch(
	ctx context.Context,
	req *GenerateBatchRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.SettlementBatch, error) {
	if err := requireActor(actor, "Settlement batch generation"); err != nil {
		return nil, err
	}
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	bounds := PeriodBounds{PeriodStart: req.PeriodStart, PeriodEnd: req.PeriodEnd}
	if bounds.PeriodStart == 0 || bounds.PeriodEnd == 0 {
		bounds = ResolveCurrentPeriod(control, now)
	} else {
		bounds.PayDate = time.Unix(bounds.PeriodEnd, 0).UTC().
			AddDate(0, 0, control.PayDelayDays).Unix()
	}
	if bounds.PeriodEnd <= bounds.PeriodStart {
		return nil, errortypes.NewValidationError(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}

	batch, err := s.resolveOpenBatch(ctx, req, bounds, actor, now)
	if err != nil {
		return nil, err
	}

	workerIDs, err := s.payEventRepo.ListWorkerIDsWithAccruedEvents(
		ctx,
		repositories.ListWorkersWithAccruedEventsRequest{
			TenantInfo: req.TenantInfo,
			PeriodEnd:  bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}

	var generated int
	for _, workerID := range workerIDs {
		settlement, genErr := s.GenerateForWorker(ctx, &GenerateForWorkerRequest{
			TenantInfo:  req.TenantInfo,
			WorkerID:    workerID,
			PeriodStart: bounds.PeriodStart,
			PeriodEnd:   bounds.PeriodEnd,
			PayDate:     bounds.PayDate,
			BatchID:     &batch.ID,
		}, actor)
		if genErr != nil {
			s.l.Error("failed to generate settlement for worker",
				zap.Error(genErr), zap.String("workerId", workerID.String()))
			continue
		}
		if settlement != nil {
			generated++
		}
	}

	batch, err = s.batchRepo.RecalculateAggregates(ctx, req.TenantInfo, batch.ID)
	if err != nil {
		return nil, err
	}

	s.l.Info("settlement batch generated",
		zap.String("batchId", batch.ID.String()),
		zap.Int("newSettlements", generated),
		zap.Int("settlements", batch.SettlementCount),
		zap.Int("exceptions", batch.ExceptionCount))
	return batch, nil
}

func (s *Service) resolveOpenBatch(
	ctx context.Context,
	req *GenerateBatchRequest,
	bounds PeriodBounds,
	actor *serviceports.RequestActor,
	now int64,
) (*driversettlement.SettlementBatch, error) {
	existing, err := s.batchRepo.GetForPeriod(
		ctx,
		req.TenantInfo,
		bounds.PeriodStart,
		bounds.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.Status != driversettlement.BatchStatusOpen {
			return nil, errortypes.NewValidationError(
				"periodEnd",
				errortypes.ErrInvalidOperation,
				"The batch for this pay period is already completed; generate individual settlements for late accruals instead",
			)
		}
		return existing, nil
	}

	name := req.Name
	if name == "" {
		name = "Pay period ending " +
			time.Unix(bounds.PeriodEnd, 0).UTC().AddDate(0, 0, -1).Format("Jan 2, 2006")
	}

	batch := &driversettlement.SettlementBatch{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         driversettlement.BatchStatusOpen,
		Name:           name,
		PeriodStart:    bounds.PeriodStart,
		PeriodEnd:      bounds.PeriodEnd,
		PayDate:        bounds.PayDate,
		Notes:          req.Notes,
		GeneratedByID:  actor.UserID,
		GeneratedAt:    &now,
	}
	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.batchRepo.Create(ctx, batch)
}

func (s *Service) GenerateOffCycle(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	hasBatch := req.BatchID != nil && !req.BatchID.IsNil()
	if hasBatch {
		batch, err := s.batchRepo.GetByID(ctx, repositories.GetSettlementBatchByIDRequest{
			ID:         *req.BatchID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if batch.Status != driversettlement.BatchStatusOpen {
			return nil, errortypes.NewValidationError(
				"batchId",
				errortypes.ErrInvalidOperation,
				"Settlements can only be added to an open batch",
			)
		}
	}

	settlement, err := s.GenerateForWorker(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	if settlement != nil && hasBatch {
		if _, aggErr := s.batchRepo.RecalculateAggregates(
			ctx,
			req.TenantInfo,
			*req.BatchID,
		); aggErr != nil {
			s.l.Error("failed to refresh batch aggregates after off-cycle settlement",
				zap.Error(aggErr))
		}
	}
	return settlement, nil
}

type GenerateForWorkerRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	PeriodStart int64
	PeriodEnd   int64
	PayDate     int64
	BatchID     *pulid.ID

	// PayEventIDs restricts the settlement to an explicit set of the worker's
	// accrued events instead of everything accrued through PeriodEnd.
	PayEventIDs []pulid.ID
	// SkipRecurring leaves out period-cycle items — recurring earnings and
	// deductions, advance recovery, guarantee top-up, carry-forward, and
	// variance checks — so an instant payout only covers the selected loads.
	SkipRecurring bool
}

func (s *Service) GenerateForWorker(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement generation"); err != nil {
		return nil, err
	}
	if len(req.PayEventIDs) == 0 {
		exists, existsErr := s.settlementRepo.ExistsForWorkerPeriod(
			ctx,
			req.TenantInfo,
			req.WorkerID,
			req.PeriodStart,
			req.PeriodEnd,
		)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, nil //nolint:nilnil // nil settlement means this worker period is already settled
		}
	}

	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	var created *driversettlement.Settlement
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		settlement, events, buildErr := s.buildSettlement(txCtx, req, control)
		if buildErr != nil {
			return buildErr
		}
		if settlement == nil {
			return nil
		}

		number, seqErr := s.generator.GenerateDriverSettlementNumber(
			txCtx,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			"",
			"",
		)
		if seqErr != nil {
			return seqErr
		}
		settlement.SettlementNumber = number

		var txErr error
		created, txErr = s.settlementRepo.Create(txCtx, settlement)
		if txErr != nil {
			return txErr
		}

		eventIDs := make([]pulid.ID, 0, len(events))
		for _, event := range events {
			eventIDs = append(eventIDs, event.ID)
		}
		return s.payEventRepo.MarkSettled(txCtx, req.TenantInfo, eventIDs, created.ID)
	})
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, nil //nolint:nilnil // nil settlement means the worker had no accrued pay
	}

	s.logSettlementAudit(ctx, created, nil, actor.UserID, permission.OpCreate,
		"Settlement generated for period")

	if control.AutoApproveClean && !created.HasExceptions {
		approved, approveErr := s.approveInternal(ctx, req.TenantInfo, created.ID, actor, true)
		if approveErr != nil {
			s.l.Error("failed to auto-approve clean settlement",
				zap.Error(approveErr), zap.String("settlementId", created.ID.String()))
			return created, nil
		}
		return approved, nil
	}
	return created, nil
}

func (s *Service) PreviewForWorker(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) (*driversettlement.Settlement, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if req.PeriodStart == 0 || req.PeriodEnd == 0 {
		bounds := ResolveCurrentPeriod(control, timeutils.NowUnix())
		req.PeriodStart = bounds.PeriodStart
		req.PeriodEnd = bounds.PeriodEnd
		req.PayDate = bounds.PayDate
	}
	settlement, _, err := s.buildSettlement(ctx, req, control)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker has no accrued pay events for this period",
		)
	}
	settlement.SettlementNumber = "PREVIEW"
	return settlement, nil
}

func (s *Service) buildSettlement(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	control *tenant.SettlementControl,
) (*driversettlement.Settlement, []*driversettlement.PayEvent, error) {
	events, err := s.payEventRepo.ListAccruedForWorker(
		ctx,
		&repositories.ListAccruedPayEventsRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			PeriodEnd:  req.PeriodEnd,
			EventIDs:   req.PayEventIDs,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if len(req.PayEventIDs) > 0 && len(events) != len(req.PayEventIDs) {
		return nil, nil, errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Some selected pay events are no longer payable — they may be settled, held, or voided. Refresh and try again.",
		)
	}
	if len(events) == 0 {
		return nil, nil, nil
	}

	settlement := &driversettlement.Settlement{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		BatchID:        req.BatchID,
		Status:         driversettlement.StatusDraft,
		Classification: driverpay.PayeeClassificationCompanyDriver,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		PayDate:        req.PayDate,
		CurrencyCode:   money.DefaultCurrencyCode,
	}

	assignment, assignErr := s.assignmentRepo.GetEffectiveForWorker(
		ctx,
		repositories.GetWorkerPayAssignmentRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			AsOf:       req.PeriodEnd,
		},
	)
	var profile *driverpay.PayProfile
	if assignErr == nil && assignment != nil {
		profile = assignment.PayProfile
		profileID := assignment.PayProfileID
		settlement.PayProfileID = &profileID
		if profile != nil {
			settlement.PayProfileName = profile.Name
			settlement.CurrencyCode = profile.CurrencyCode
		}
	} else {
		settlement.AddException(
			driversettlement.ExceptionCodeMissingPayProfile,
			driversettlement.ExceptionSeverityCritical,
			"Worker has no effective pay profile assignment for this period",
		)
	}
	settlement.Classification = classificationForWorker(profile)

	lines, totalMiles, shipmentCount := buildEventEarningLines(events)
	settlement.ShipmentCount = shipmentCount
	settlement.TotalMiles = totalMiles.Round(2)

	if !req.SkipRecurring {
		lines, err = s.appendRecurringCycleLines(ctx, req, control, settlement, profile, lines)
		if err != nil {
			return nil, nil, err
		}
	}

	settlement.Lines = lines
	settlement.SyncTotals()

	if settlement.CarryForwardOutMinor < 0 {
		settlement.AddException(
			driversettlement.ExceptionCodeNegativeNet,
			driversettlement.ExceptionSeverityCritical,
			fmt.Sprintf("Deductions exceed earnings; %s carries forward",
				money.FormatMinor(-settlement.CarryForwardOutMinor, settlement.CurrencyCode)),
		)
	}

	if !req.SkipRecurring {
		if err = s.flagVarianceException(ctx, req, control, settlement); err != nil {
			return nil, nil, err
		}
	}

	return settlement, events, nil
}

// appendRecurringCycleLines adds everything tied to the pay-period cycle —
// recurring earnings, guaranteed minimum, carry-forward, recurring deductions,
// and advance recovery. Instant payouts skip all of it.
func (s *Service) appendRecurringCycleLines(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	control *tenant.SettlementControl,
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

func buildEventEarningLines(
	events []*driversettlement.PayEvent,
) (lines []*driversettlement.SettlementLine, totalMiles decimal.Decimal, shipmentCount int) {
	lines = make([]*driversettlement.SettlementLine, 0, len(events)*2)
	totalMiles = decimal.Zero
	shipments := make(map[pulid.ID]struct{}, len(events))
	for _, event := range events {
		shipments[event.ShipmentID] = struct{}{}
		totalMiles = totalMiles.Add(event.TotalMiles)
		eventID := event.ID
		shipmentID := event.ShipmentID
		for _, comp := range event.Components {
			lines = append(lines, &driversettlement.SettlementLine{
				Category:      driversettlement.LineCategoryEarning,
				ComponentKind: comp.Kind,
				Method:        comp.Method,
				Description:   comp.Description,
				Quantity:      comp.Quantity,
				Rate:          comp.Rate,
				AmountMinor:   comp.AmountMinor,
				ShipmentID:    &shipmentID,
				MoveID:        event.MoveID,
				PayEventID:    &eventID,
				ProNumber:     event.ProNumber,
			})
		}
	}
	return lines, totalMiles, len(shipments)
}

func (s *Service) resolveCarryForward(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) int64 {
	latest, err := s.settlementRepo.GetLatestForWorker(
		ctx,
		repositories.GetLatestSettlementForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil || latest == nil {
		return 0
	}
	return latest.CarryForwardOutMinor
}

func (s *Service) appendEarningLines(
	ctx context.Context,
	req *GenerateForWorkerRequest,
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

func (s *Service) buildEarningLines(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) (lines []*driversettlement.SettlementLine, capped bool, guaranteeExcluded int64, err error) {
	earnings, err := s.earningRepo.ListActiveForWorker(
		ctx,
		repositories.ListActiveEarningsForWorkerRequest{
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
		if earning.Frequency == driverpay.EarningFrequencyMonthly &&
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

func (s *Service) buildDeductionLines(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) ([]*driversettlement.SettlementLine, bool, error) {
	deductions, err := s.deductionRepo.ListActiveForWorker(
		ctx,
		repositories.ListActiveDeductionsForWorkerRequest{
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
		if deduction.Frequency == driverpay.DeductionFrequencyMonthly &&
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
				repositories.GetActiveEscrowAccountForWorkerRequest{
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

func (s *Service) monthlyRecurrenceApplies(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) bool {
	latest, err := s.settlementRepo.GetLatestForWorker(
		ctx,
		repositories.GetLatestSettlementForWorkerRequest{
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

func (s *Service) buildAdvanceLines(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	control *tenant.SettlementControl,
	gross int64,
	existingLines []*driversettlement.SettlementLine,
) ([]*driversettlement.SettlementLine, error) {
	advances, err := s.advanceRepo.ListOutstandingForWorker(
		ctx,
		repositories.ListOutstandingAdvancesForWorkerRequest{
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

func (s *Service) flagVarianceException(
	ctx context.Context,
	req *GenerateForWorkerRequest,
	control *tenant.SettlementControl,
	settlement *driversettlement.Settlement,
) error {
	if control.VarianceThresholdPct.IsZero() {
		return nil
	}
	nets, err := s.settlementRepo.ListTrailingNetPay(
		ctx,
		&repositories.ListTrailingNetPayRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			Limit:      control.VarianceLookbackWeeks,
			BeforeDate: req.PeriodEnd,
		},
	)
	if err != nil {
		return err
	}
	if len(nets) < 2 {
		return nil
	}
	var sum int64
	for _, net := range nets {
		sum += net
	}
	average := sum / int64(len(nets))
	if average <= 0 {
		return nil
	}
	diff := settlement.NetPayMinor - average
	if diff < 0 {
		diff = -diff
	}
	variancePct := decimal.NewFromInt(diff).
		Div(decimal.NewFromInt(average)).
		Mul(decimal.NewFromInt(100))
	if variancePct.GreaterThanOrEqual(control.VarianceThresholdPct) {
		settlement.AddException(
			driversettlement.ExceptionCodeHighVariance,
			driversettlement.ExceptionSeverityWarning,
			fmt.Sprintf(
				"Net pay deviates %s%% from the trailing %d-settlement average of %s",
				variancePct.Round(1).String(),
				len(nets),
				money.FormatMinor(average, settlement.CurrencyCode),
			),
		)
	}
	return nil
}
