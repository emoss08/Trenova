package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) Post(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *permissionports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := settlementshared.RequireActor(actor, "Carrier settlement posting"); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		previous = *entity
		now := timeutils.NowUnix()
		if txErr = PlanPost(entity, actor.UserID, now); txErr != nil {
			return txErr
		}
		if txErr = s.stamper.StampInto(txCtx, &exchangeratestamp.Request{
			TenantInfo:     tenantInfo,
			CurrencyCode:   entity.CurrencyCode,
			DocumentDate:   now,
			AccountingDate: now,
		}, &entity.ExchangeRate, &entity.ExchangeRateDate); txErr != nil {
			return txErr
		}

		batchID, txErr := s.journals.PostSettlement(txCtx, entity, actor, false)
		if txErr != nil {
			return txErr
		}

		if txErr = s.appendLedgerEntry(txCtx, entity, &ledgerEntryParams{
			EntryType:       carriersettlement.LedgerEntryTypeBill,
			SourceEvent:     accountingcontrol.JournalSourceEventCarrierSettlementPosted,
			JournalBatchID:  batchID,
			AmountMinor:     entity.NetPayableMinor,
			TransactionDate: now,
			Actor:           actor,
		}); txErr != nil {
			return txErr
		}
		if txErr = s.costEventRepo.MarkSettled(txCtx, tenantInfo, entity.ID); txErr != nil {
			return txErr
		}

		entity.PostedJournalBatchID = batchID
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		return s.queueSync(
			txCtx,
			updated,
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncOperationCreate,
			accountingsync.SyncSourceCarrierSettlementPosted,
		)
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Carrier settlement posted to general ledger")
	return updated, nil
}

func (s *Service) postVoidReversal(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *permissionports.RequestActor,
) error {
	batchID, err := s.journals.PostSettlement(ctx, entity, actor, true)
	if err != nil {
		return err
	}
	entity.VoidJournalBatchID = batchID
	if entity.NetPayableMinor == 0 {
		return nil
	}
	return s.appendLedgerEntry(ctx, entity, &ledgerEntryParams{
		EntryType:       carriersettlement.LedgerEntryTypeAdjustment,
		SourceEvent:     accountingcontrol.JournalSourceEventCarrierSettlementVoided,
		JournalBatchID:  batchID,
		AmountMinor:     -entity.NetPayableMinor,
		TransactionDate: timeutils.NowUnix(),
		Actor:           actor,
	})
}

func (s *Service) appendLedgerEntry(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	params *ledgerEntryParams,
) error {
	if params.AmountMinor == 0 {
		return nil
	}
	settlementID := entity.ID
	entry := &carriersettlement.LedgerEntry{
		OrganizationID:      entity.OrganizationID,
		BusinessUnitID:      entity.BusinessUnitID,
		CarrierID:           entity.CarrierID,
		EntryType:           params.EntryType,
		SourceObjectType:    "CarrierSettlement",
		SourceObjectID:      entity.ID.String(),
		SourceEventType:     params.SourceEvent.String(),
		RelatedSettlementID: &settlementID,
		JournalBatchID:      params.JournalBatchID,
		DocumentNumber:      entity.SettlementNumber,
		TransactionDate:     params.TransactionDate,
		LineNumber:          1,
		AmountMinor:         params.AmountMinor,
		CreatedByID:         params.Actor.UserID,
	}
	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return s.ledgerRepo.AppendEntries(ctx, []*carriersettlement.LedgerEntry{entry})
}
