package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"
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
		"Cannot transition carrier settlement from {0} to {1}", from.String(), to.String(),
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
	previous := *entity
	if err = PlanSubmit(entity, actor.UserID, timeutils.NowUnix()); err != nil {
		return nil, err
	}

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
			if txErr = PlanSubmit(entity, actor.UserID, timeutils.NowUnix()); txErr != nil {
				return txErr
			}
			entity, txErr = s.settlementRepo.Update(txCtx, entity)
			if txErr != nil {
				return txErr
			}
		}

		previous = *entity
		if txErr = PlanApprove(entity, actor.UserID, timeutils.NowUnix()); txErr != nil {
			return txErr
		}
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
	previous := *entity
	if err = PlanReject(entity, reason); err != nil {
		return nil, err
	}

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
	req *serviceports.MarkSettlementPaidRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement payment"); err != nil {
		return nil, err
	}
	paymentMethod := req.PaymentMethod
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
		entity, txErr := s.getForUpdate(txCtx, req.TenantInfo, req.SettlementID)
		if txErr != nil {
			return txErr
		}
		if txErr = CheckMarkPaid(entity, paymentMethod); txErr != nil {
			return txErr
		}
		previous = *entity

		paidAt := req.PaidAt
		if paidAt == 0 {
			paidAt = timeutils.NowUnix()
		}
		if txErr = s.stamper.StampInto(txCtx, &exchangeratestamp.Request{
			TenantInfo:     req.TenantInfo,
			CurrencyCode:   entity.CurrencyCode,
			DocumentDate:   paidAt,
			AccountingDate: paidAt,
		}, &entity.PaidExchangeRate, &entity.PaidExchangeRateDate); txErr != nil {
			return txErr
		}
		batchID, txErr := s.postPaymentJournal(txCtx, entity, actor, paidAt)
		if txErr != nil {
			return txErr
		}

		if txErr = s.appendLedgerEntry(txCtx, entity, &ledgerEntryParams{
			EntryType:       carriersettlement.LedgerEntryTypePayment,
			SourceEvent:     tenant.JournalSourceEventCarrierSettlementPaid,
			JournalBatchID:  batchID,
			AmountMinor:     -entity.NetPayableMinor,
			TransactionDate: paidAt,
			Actor:           actor,
		}); txErr != nil {
			return txErr
		}

		if txErr = PlanMarkPaid(entity, &MarkPaidInput{
			PaymentMethod:    paymentMethod,
			PaymentReference: req.PaymentReference,
			PaidAt:           paidAt,
			UserID:           actor.UserID,
		}); txErr != nil {
			return txErr
		}
		entity.PaidJournalBatchID = batchID
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		if updated.NetPayableMinor == 0 {
			return nil
		}
		return s.queueSync(
			txCtx,
			updated,
			accountingsync.SyncObjectCarrierBillPay,
			accountingsync.SyncOperationCreate,
			accountingsync.SyncSourceCarrierSettlementPaid,
		)
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
		if txErr = CheckVoid(entity, reason); txErr != nil {
			return txErr
		}
		previous = *entity

		wasPosted := entity.Status == carriersettlement.StatusPosted
		if wasPosted {
			if txErr = s.postVoidReversal(txCtx, entity, actor); txErr != nil {
				return txErr
			}
		}

		if txErr = s.costEventRepo.ReleaseSettled(txCtx, tenantInfo, entity.ID); txErr != nil {
			return txErr
		}

		if txErr = PlanVoid(entity, reason, actor.UserID, timeutils.NowUnix()); txErr != nil {
			return txErr
		}
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		if !wasPosted {
			return nil
		}
		return s.queueSync(
			txCtx,
			updated,
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationVoid,
			accountingsync.SyncSourceCarrierSettlementVoided,
		)
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
		if txErr = PlanRecalculate(entity); txErr != nil {
			return txErr
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
		return nil, errNoPendingCostEventsRemain()
	}
	MergeRebuilt(entity, rebuilt)

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
	if err := checkAdjustmentInput(input); err != nil {
		return nil, err
	}

	entity, err := s.getForUpdate(ctx, tenantInfo, settlementID)
	if err != nil {
		return nil, err
	}

	previous := *entity
	if err = PlanAddAdjustment(entity, input); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.settlementRepo.ReplaceLines(txCtx, entity); txErr != nil {
			return txErr
		}
		var txErr error
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
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
	previous := *entity
	if err = PlanRemoveAdjustment(entity, lineID); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.settlementRepo.ReplaceLines(txCtx, entity); txErr != nil {
			return txErr
		}
		var txErr error
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Manual adjustment removed")
	return updated, nil
}
