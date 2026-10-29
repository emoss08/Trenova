package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) Post(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := settlementshared.RequireActor(actor, "Settlement posting"); err != nil {
		return nil, err
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, txErr := s.GetForUpdate(txCtx, tenantInfo, settlementID)
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
		entity.PostedJournalBatchID = batchID
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		return s.queueSync(
			txCtx,
			updated,
			accountingsync.SyncObjectDriverBill,
			accountingsync.SyncOperationCreate,
			accountingsync.SyncSourceDriverSettlementPosted,
		)
	})
	if err != nil {
		return nil, err
	}
	s.LogSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Settlement posted to general ledger")
	if s.driverNotify != nil {
		s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: tenantInfo,
			WorkerID:   updated.WorkerID,
			EventType:  "dash.settlement_posted",
			Priority:   notification.PriorityHigh,
			Context: notificationtemplate.SettlementNotificationContext{
				SettlementNumber: updated.SettlementNumber,
				Amount: money.FormatMinor(
					updated.NetPayMinor, updated.CurrencyCode,
				),
				Currency: updated.CurrencyCode,
			},
			Link: "/dash/pay/" + updated.ID.String(),
			RelatedEntities: map[string]any{
				"settlementId": updated.ID.String(),
			},
		})
	}
	return updated, nil
}
