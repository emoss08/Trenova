package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func transitionError(from, to carriersettlement.Status) error {
	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Cannot transition carrier settlement from "+from.String()+" to "+to.String(),
	)
}

func (s *Service) getForUpdate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) (*carriersettlement.CarrierSettlement, error) {
	return s.settlementRepo.GetByID(ctx, repositories.GetCarrierSettlementByIDRequest{
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
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement submission"); err != nil {
		return nil, err
	}
	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}
	if !carriersettlement.IsAllowedTransition(
		entity.Status,
		carriersettlement.StatusPendingApproval,
	) || entity.Status == carriersettlement.StatusPendingApproval {
		return nil, transitionError(entity.Status, carriersettlement.StatusPendingApproval)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = carriersettlement.StatusPendingApproval
	entity.SubmittedByID = actor.UserID
	entity.SubmittedAt = &now

	updated, err := s.settlementRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpSubmit,
		"Carrier settlement submitted for approval")
	return updated, nil
}

func (s *Service) Approve(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	return s.approveInternal(ctx, tenantInfo, settlementID, actor, false)
}

func (s *Service) approveInternal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
	fromDraft bool,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement approval"); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if fromDraft && entity.Status == carriersettlement.StatusDraft {
			now := timeutils.NowUnix()
			entity.Status = carriersettlement.StatusPendingApproval
			entity.SubmittedByID = actor.UserID
			entity.SubmittedAt = &now
			entity, txErr = s.settlementRepo.Update(txCtx, entity)
			if txErr != nil {
				return txErr
			}
		}
		if entity.Status != carriersettlement.StatusPendingApproval {
			return transitionError(entity.Status, carriersettlement.StatusApproved)
		}

		previous = *entity
		now := timeutils.NowUnix()
		entity.Status = carriersettlement.StatusApproved
		entity.ApprovedByID = actor.UserID
		entity.ApprovedAt = &now
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Carrier settlement approved")

	control, controlErr := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if controlErr == nil && control.AutoPostOnApprove {
		posted, postErr := s.Post(ctx, tenantInfo, settlementID, actor)
		if postErr != nil {
			s.l.Error("failed to auto-post carrier settlement after approval",
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
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement rejection"); err != nil {
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
	if entity.Status != carriersettlement.StatusPendingApproval {
		return nil, transitionError(entity.Status, carriersettlement.StatusDraft)
	}

	previous := *entity
	entity.Status = carriersettlement.StatusDraft
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
		"Carrier settlement rejected: "+reason)
	return updated, nil
}

// MarkPaid transitions a posted settlement to paid and, unlike its driver
// counterpart, also posts the cash disbursement — DR accounts payable, CR the
// accounting control's default cash account — and records the Payment entry in
// the carrier ledger so AP credited at posting time actually clears.
func (s *Service) MarkPaid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	paymentMethod, paymentReference string,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement payment"); err != nil {
		return nil, err
	}
	if paymentMethod == "" {
		return nil, errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required",
		)
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != carriersettlement.StatusPosted {
			return transitionError(entity.Status, carriersettlement.StatusPaid)
		}
		previous = *entity

		batchID, txErr := s.postPaymentJournal(txCtx, entity, actor)
		if txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		if txErr = s.appendLedgerEntry(txCtx, entity, &ledgerEntryParams{
			EntryType:       carriersettlement.LedgerEntryTypePayment,
			SourceEvent:     tenant.JournalSourceEventCarrierSettlementPaid,
			JournalBatchID:  batchID,
			AmountMinor:     -entity.NetPayableMinor,
			TransactionDate: now,
			Actor:           actor,
		}); txErr != nil {
			return txErr
		}

		entity.Status = carriersettlement.StatusPaid
		entity.PaidAt = &now
		entity.PaidByID = actor.UserID
		entity.PaymentMethod = paymentMethod
		entity.PaymentReference = paymentReference
		entity.PaidJournalBatchID = batchID
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Carrier settlement marked paid via "+paymentMethod)
	return updated, nil
}

func (s *Service) Void(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	reason string,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement void"); err != nil {
		return nil, err
	}
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A void reason is required",
		)
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if !carriersettlement.IsAllowedTransition(
			entity.Status,
			carriersettlement.StatusVoided,
		) || entity.Status == carriersettlement.StatusVoided {
			return transitionError(entity.Status, carriersettlement.StatusVoided)
		}
		previous = *entity

		if entity.Status == carriersettlement.StatusPosted {
			if txErr = s.postVoidReversal(txCtx, entity, actor); txErr != nil {
				return txErr
			}
		}

		if txErr = s.costEventRepo.ReleaseSettled(txCtx, tenantInfo, entity.ID); txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		entity.Status = carriersettlement.StatusVoided
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
		"Carrier settlement voided: "+reason)
	return updated, nil
}

func (s *Service) Recalculate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement recalculation"); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != carriersettlement.StatusDraft {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only draft carrier settlements can be recalculated",
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
		"Carrier settlement recalculated")
	return updated, nil
}

func (s *Service) rebuildDraftSettlementTx(
	txCtx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *carriersettlement.CarrierSettlement,
) (*carriersettlement.CarrierSettlement, error) {
	if err := s.costEventRepo.ReleaseSettled(txCtx, tenantInfo, entity.ID); err != nil {
		return nil, err
	}

	rebuilt, events, err := s.buildSettlement(txCtx, &GenerateForCarrierRequest{
		TenantInfo:  tenantInfo,
		CarrierID:   entity.CarrierID,
		PeriodStart: entity.PeriodStart,
		PeriodEnd:   entity.PeriodEnd,
		PayDate:     entity.PayDate,
		BatchID:     entity.BatchID,
	})
	if err != nil {
		return nil, err
	}
	if rebuilt == nil {
		return nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalid,
			"No pending cost events remain for this settlement's period",
		)
	}

	manualLines := make([]*carriersettlement.CarrierSettlementLine, 0)
	for _, line := range entity.Lines {
		if line != nil && line.IsManualAdjustment() {
			line.ID = pulid.Nil
			manualLines = append(manualLines, line)
		}
	}

	mergedLines := make(
		[]*carriersettlement.CarrierSettlementLine,
		0,
		len(rebuilt.Lines)+len(manualLines),
	)
	mergedLines = append(mergedLines, rebuilt.Lines...)
	mergedLines = append(mergedLines, manualLines...)
	entity.Lines = mergedLines
	entity.CurrencyCode = rebuilt.CurrencyCode
	entity.ShipmentCount = rebuilt.ShipmentCount
	entity.SyncTotals()

	if err = s.settlementRepo.ReplaceLines(txCtx, entity); err != nil {
		return nil, err
	}
	eventIDs := make([]pulid.ID, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}
	if err = s.costEventRepo.MarkAttached(txCtx, tenantInfo, eventIDs, entity.ID); err != nil {
		return nil, err
	}
	return s.settlementRepo.Update(txCtx, entity)
}

type AdjustmentLineInput struct {
	Description string
	AmountMinor int64
	GLAccountID *pulid.ID
}

func (s *Service) AddAdjustmentLine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	input *AdjustmentLineInput,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement adjustment"); err != nil {
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
			"Only draft or pending carrier settlements can be adjusted",
		)
	}

	var glAccountID *pulid.ID
	if input.GLAccountID != nil && !input.GLAccountID.IsNil() {
		glAccountID = input.GLAccountID
	}

	previous := *entity
	entity.Lines = append(entity.Lines, &carriersettlement.CarrierSettlementLine{
		EventType:   carriersettlement.CostEventTypeAdjustment,
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		GLAccountID: glAccountID,
	})
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
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement adjustment removal"); err != nil {
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
			"Only draft or pending carrier settlements can be adjusted",
		)
	}

	previous := *entity
	found := false
	remaining := make([]*carriersettlement.CarrierSettlementLine, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.ID == lineID {
			if !line.IsManualAdjustment() {
				return nil, errortypes.NewValidationError(
					"lineId",
					errortypes.ErrInvalidOperation,
					"Only manual adjustment lines can be removed",
				)
			}
			found = true
			continue
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
