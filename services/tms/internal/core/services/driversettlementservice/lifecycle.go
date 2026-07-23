package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func transitionError(from, to driversettlement.Status) error {
	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Cannot transition settlement from "+from.String()+" to "+to.String(),
	)
}

func (s *Service) getForUpdate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) (*driversettlement.Settlement, error) {
	return s.settlementRepo.GetByID(ctx, repositories.GetDriverSettlementByIDRequest{
		ID:           settlementID,
		TenantInfo:   tenantInfo,
		IncludeLines: true,
	})
}

func (s *Service) SubmitForApproval(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement submission"); err != nil {
		return nil, err
	}
	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if !driversettlement.IsAllowedTransition(
		entity.Status,
		driversettlement.StatusPendingApproval,
	) || entity.Status == driversettlement.StatusPendingApproval {
		return nil, transitionError(entity.Status, driversettlement.StatusPendingApproval)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = driversettlement.StatusPendingApproval
	entity.SubmittedByID = actor.UserID
	entity.SubmittedAt = &now

	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpSubmit,
		"Settlement submitted for approval")
	return updated, nil
}

func (s *Service) Approve(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	return s.approveInternal(ctx, tenantInfo, settlementID, actor, false)
}

func (s *Service) approveInternal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
	fromDraft bool,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement approval"); err != nil {
		return nil, err
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if fromDraft && entity.Status == driversettlement.StatusDraft {
			now := timeutils.NowUnix()
			entity.Status = driversettlement.StatusPendingApproval
			entity.SubmittedByID = actor.UserID
			entity.SubmittedAt = &now
			entity, txErr = s.settlementRepo.Update(txCtx, entity)
			if txErr != nil {
				return txErr
			}
		}
		if entity.Status != driversettlement.StatusPendingApproval {
			return transitionError(entity.Status, driversettlement.StatusApproved)
		}

		previous = *entity
		if txErr = s.applyDeductionSideEffects(txCtx, entity, actor); txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		entity.Status = driversettlement.StatusApproved
		entity.ApprovedByID = actor.UserID
		entity.ApprovedAt = &now
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Settlement approved")

	control, controlErr := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if controlErr == nil && control.AutoPostOnApprove {
		posted, postErr := s.Post(ctx, tenantInfo, settlementID, actor)
		if postErr != nil {
			s.l.Error("failed to auto-post settlement after approval",
				zap.Error(postErr), zap.String("settlementId", settlementID.String()))
			return updated, nil
		}
		return posted, nil
	}
	return updated, nil
}

func (s *Service) Reject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	reason string,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement rejection"); err != nil {
		return nil, err
	}
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A rejection reason is required",
		)
	}
	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if entity.Status != driversettlement.StatusPendingApproval {
		return nil, transitionError(entity.Status, driversettlement.StatusDraft)
	}

	previous := *entity
	entity.Status = driversettlement.StatusDraft
	entity.SubmittedByID = pulid.Nil
	entity.SubmittedAt = nil
	if entity.Notes != "" {
		entity.Notes += "\n"
	}
	entity.Notes += "Rejected: " + reason

	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpReject,
		"Settlement rejected: "+reason)
	return updated, nil
}

func (s *Service) MarkPaid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	paymentMethod, paymentReference string,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement payment"); err != nil {
		return nil, err
	}
	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if entity.Status != driversettlement.StatusPosted {
		return nil, transitionError(entity.Status, driversettlement.StatusPaid)
	}
	if paymentMethod == "" {
		return nil, errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required",
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = driversettlement.StatusPaid
	entity.PaidAt = &now
	entity.PaidByID = actor.UserID
	entity.PaymentMethod = paymentMethod
	entity.PaymentReference = paymentReference

	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Settlement marked paid via "+paymentMethod)
	if s.driverNotify != nil {
		s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: tenantInfo,
			WorkerID:   updated.WorkerID,
			EventType:  "dash.settlement_paid",
			Priority:   notification.PriorityHigh,
			Title:      "You've been paid",
			Message:    "Settlement " + updated.SettlementNumber + " was paid via " + paymentMethod + ".",
			Link:       "/dash/pay/" + updated.ID.String(),
			RelatedEntities: map[string]any{
				"settlementId": updated.ID.String(),
			},
		})
	}
	return updated, nil
}

func (s *Service) Void(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	reason string,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement void"); err != nil {
		return nil, err
	}
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A void reason is required",
		)
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if !driversettlement.IsAllowedTransition(
			entity.Status,
			driversettlement.StatusVoided,
		) || entity.Status == driversettlement.StatusVoided {
			return transitionError(entity.Status, driversettlement.StatusVoided)
		}
		previous = *entity

		wasApproved := entity.Status == driversettlement.StatusApproved ||
			entity.Status == driversettlement.StatusPosted
		if wasApproved {
			if txErr = s.reverseDeductionSideEffects(txCtx, entity, actor); txErr != nil {
				return txErr
			}
		}
		if entity.Status == driversettlement.StatusPosted {
			if txErr = s.postVoidReversal(txCtx, entity, actor); txErr != nil {
				return txErr
			}
		}

		if txErr = s.payEventRepo.ReleaseSettled(txCtx, tenantInfo, entity.ID); txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		entity.Status = driversettlement.StatusVoided
		entity.VoidedByID = actor.UserID
		entity.VoidedAt = &now
		entity.VoidReason = reason
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpCancel,
		"Settlement voided: "+reason)
	return updated, nil
}

func (s *Service) Recalculate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement recalculation"); err != nil {
		return nil, err
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != driversettlement.StatusDraft {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only draft settlements can be recalculated",
			)
		}
		previous = *entity

		updated, txErr = s.rebuildDraftSettlementTx(txCtx, tenantInfo, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Settlement recalculated")
	return updated, nil
}

func (s *Service) rebuildDraftSettlementTx(
	txCtx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	if err := s.payEventRepo.ReleaseSettled(txCtx, tenantInfo, entity.ID); err != nil {
		return nil, err
	}

	control, err := s.settlementControl.GetOrCreate(txCtx, tenantInfo)
	if err != nil {
		return nil, err
	}
	rebuilt, events, err := s.buildSettlement(txCtx, &GenerateForWorkerRequest{
		TenantInfo:  tenantInfo,
		WorkerID:    entity.WorkerID,
		PeriodStart: entity.PeriodStart,
		PeriodEnd:   entity.PeriodEnd,
		PayDate:     entity.PayDate,
		BatchID:     entity.BatchID,
	}, control)
	if err != nil {
		return nil, err
	}
	if rebuilt == nil {
		return nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalid,
			"No accrued pay events remain for this settlement's period",
		)
	}

	manualLines := make([]*driversettlement.SettlementLine, 0)
	for _, line := range entity.Lines {
		if line != nil && line.Category == driversettlement.LineCategoryAdjustment {
			line.ID = pulid.Nil
			manualLines = append(manualLines, line)
		}
	}

	mergedLines := make(
		[]*driversettlement.SettlementLine,
		0,
		len(rebuilt.Lines)+len(manualLines),
	)
	mergedLines = append(mergedLines, rebuilt.Lines...)
	mergedLines = append(mergedLines, manualLines...)
	entity.Lines = mergedLines
	entity.ClearExceptions()
	entity.Exceptions = rebuilt.Exceptions
	entity.HasExceptions = rebuilt.HasExceptions
	entity.PayProfileID = rebuilt.PayProfileID
	entity.PayProfileName = rebuilt.PayProfileName
	entity.Classification = rebuilt.Classification
	entity.CurrencyCode = rebuilt.CurrencyCode
	entity.TotalMiles = rebuilt.TotalMiles
	entity.ShipmentCount = rebuilt.ShipmentCount
	if len(manualLines) > 0 {
		entity.AddException(
			driversettlement.ExceptionCodeManualAdjustment,
			driversettlement.ExceptionSeverityWarning,
			"Settlement contains manual adjustment lines",
		)
	}
	entity.SyncTotals()

	if err = s.settlementRepo.ReplaceLines(txCtx, entity); err != nil {
		return nil, err
	}
	eventIDs := make([]pulid.ID, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}
	if err = s.payEventRepo.MarkSettled(txCtx, tenantInfo, eventIDs, entity.ID); err != nil {
		return nil, err
	}
	return s.settlementRepo.Update(txCtx, entity)
}

func (s *Service) refreshOpenDraftForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) error {
	draft, err := s.settlementRepo.GetOpenDraftForWorker(
		ctx,
		repositories.GetOpenDraftForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err != nil || draft == nil {
		return nil //nolint:nilerr // no open draft means nothing to attach to
	}
	return s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, draft.ID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != driversettlement.StatusDraft {
			return nil
		}
		_, txErr = s.rebuildDraftSettlementTx(txCtx, tenantInfo, entity)
		return txErr
	})
}

type AdjustmentLineInput struct {
	Description string
	AmountMinor int64
	Quantity    decimal.Decimal
	Rate        decimal.Decimal
	PayCodeID   *pulid.ID
}

func (s *Service) AddAdjustmentLine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	input *AdjustmentLineInput,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement adjustment"); err != nil {
		return nil, err
	}
	if input.Description == "" {
		return nil, errortypes.NewValidationError(
			"description",
			errortypes.ErrRequired,
			"Adjustment description is required",
		)
	}
	if input.AmountMinor == 0 {
		return nil, errortypes.NewValidationError(
			"amountMinor",
			errortypes.ErrInvalid,
			"Adjustment amount cannot be zero",
		)
	}

	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if !entity.IsEditable() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft or pending settlements can be adjusted",
		)
	}

	var payCodeID *pulid.ID
	if input.PayCodeID != nil && !input.PayCodeID.IsNil() {
		if _, err = s.payCodeRepo.GetByID(ctx, repositories.GetPayCodeByIDRequest{
			ID:         *input.PayCodeID,
			TenantInfo: tenantInfo,
		}); err != nil {
			return nil, err
		}
		payCodeID = input.PayCodeID
	}

	previous := *entity
	entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
		Category:    driversettlement.LineCategoryAdjustment,
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		Quantity:    input.Quantity,
		Rate:        input.Rate,
		PayCodeID:   payCodeID,
	})
	entity.AddException(
		driversettlement.ExceptionCodeManualAdjustment,
		driversettlement.ExceptionSeverityWarning,
		"Settlement contains manual adjustment lines",
	)
	entity.SyncTotals()

	if err = s.settlementRepo.ReplaceLines(ctx, entity); err != nil {
		return nil, err
	}
	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Manual adjustment added: "+input.Description)
	return updated, nil
}

func (s *Service) RemoveAdjustmentLine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID, lineID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement adjustment removal"); err != nil {
		return nil, err
	}
	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if !entity.IsEditable() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft or pending settlements can be adjusted",
		)
	}

	previous := *entity
	found := false
	remaining := make([]*driversettlement.SettlementLine, 0, len(entity.Lines))
	hasOtherAdjustments := false
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.ID == lineID {
			if line.Category != driversettlement.LineCategoryAdjustment {
				return nil, errortypes.NewValidationError(
					"lineId",
					errortypes.ErrInvalidOperation,
					"Only manual adjustment lines can be removed",
				)
			}
			found = true
			continue
		}
		if line.Category == driversettlement.LineCategoryAdjustment {
			hasOtherAdjustments = true
		}
		remaining = append(remaining, line)
	}
	if !found {
		return nil, errortypes.NewValidationError(
			"lineId",
			errortypes.ErrInvalid,
			"Adjustment line not found on this settlement",
		)
	}

	entity.Lines = remaining
	if !hasOtherAdjustments {
		filtered := make([]driversettlement.Exception, 0, len(entity.Exceptions))
		for _, exception := range entity.Exceptions {
			if exception.Code != driversettlement.ExceptionCodeManualAdjustment {
				filtered = append(filtered, exception)
			}
		}
		entity.Exceptions = filtered
		entity.HasExceptions = len(filtered) > 0
	}
	entity.SyncTotals()

	if err = s.settlementRepo.ReplaceLines(ctx, entity); err != nil {
		return nil, err
	}
	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Manual adjustment removed")
	return updated, nil
}

func (s *Service) applyDeductionSideEffects(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *serviceports.RequestActor,
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

func (s *Service) applyRecurringEarning(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
) error {
	if line.RecurringEarningID == nil || line.RecurringEarningID.IsNil() {
		return nil
	}
	earning, err := s.earningRepo.GetByID(
		ctx,
		repositories.GetRecurringEarningByIDRequest{
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
		earning.Status = driverpay.EarningStatusCompleted
	}
	_, err = s.earningRepo.Update(ctx, earning)
	return err
}

func (s *Service) applyRecurringDeduction(
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
		repositories.GetRecurringDeductionByIDRequest{
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
		deduction.Status = driverpay.DeductionStatusCompleted
	}
	_, err = s.deductionRepo.Update(ctx, deduction)
	return err
}

func (s *Service) applyEscrowContribution(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
	actor *serviceports.RequestActor,
) error {
	if line.EscrowAccountID == nil || line.EscrowAccountID.IsNil() {
		return nil
	}
	if err := s.applyRecurringDeduction(ctx, entity, line, amount); err != nil {
		return err
	}
	return s.appendEscrowTransaction(ctx, entity, *line.EscrowAccountID,
		driverpay.EscrowTransactionTypeContribution, amount,
		"Escrow contribution from settlement "+entity.SettlementNumber, actor)
}

func (s *Service) applyAdvanceRecovery(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.AdvanceID == nil || line.AdvanceID.IsNil() {
		return nil
	}
	advance, err := s.advanceRepo.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
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

func (s *Service) reverseDeductionSideEffects(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *serviceports.RequestActor,
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
					driverpay.EscrowTransactionTypeAdjustment, -amount,
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

func (s *Service) reverseRecurringEarning(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
) error {
	if line.RecurringEarningID == nil || line.RecurringEarningID.IsNil() {
		return nil
	}
	earning, err := s.earningRepo.GetByID(
		ctx,
		repositories.GetRecurringEarningByIDRequest{
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
	if earning.Status == driverpay.EarningStatusCompleted {
		if remaining := earning.RemainingCapMinor(); remaining == nil || *remaining > 0 {
			earning.Status = driverpay.EarningStatusActive
		}
	}
	_, err = s.earningRepo.Update(ctx, earning)
	return err
}

func (s *Service) reverseRecurringDeduction(
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
		repositories.GetRecurringDeductionByIDRequest{
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
	if deduction.Status == driverpay.DeductionStatusCompleted {
		if remaining := deduction.RemainingCapMinor(); remaining == nil || *remaining > 0 {
			deduction.Status = driverpay.DeductionStatusActive
		}
	}
	_, err = s.deductionRepo.Update(ctx, deduction)
	return err
}

func (s *Service) reverseAdvanceRecovery(
	ctx context.Context,
	entity *driversettlement.Settlement,
	line *driversettlement.SettlementLine,
	amount int64,
) error {
	if line.AdvanceID == nil || line.AdvanceID.IsNil() {
		return nil
	}
	advance, err := s.advanceRepo.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
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

func (s *Service) appendEscrowTransaction(
	ctx context.Context,
	entity *driversettlement.Settlement,
	accountID pulid.ID,
	txType driverpay.EscrowTransactionType,
	amount int64,
	description string,
	actor *serviceports.RequestActor,
) error {
	account, err := s.escrowRepo.GetByID(ctx, repositories.GetEscrowAccountByIDRequest{
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
	tx := &driverpay.EscrowTransaction{
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
