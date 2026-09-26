package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *Service) HoldPayEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventID pulid.ID,
	reason string,
	actor *serviceports.RequestActor,
) (*driversettlement.PayEvent, error) {
	if err := requireActor(actor, "Pay event hold"); err != nil {
		return nil, err
	}
	if reason == "" {
		return nil, errHoldReasonRequired()
	}

	event, err := s.payEventRepo.GetByID(ctx, repositories.GetPayEventByIDRequest{
		ID:         eventID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	changed, err := PlanHoldPayEvent(event, reason)
	if err != nil {
		return nil, err
	}
	if !changed {
		return event, nil
	}

	updated, err := s.payEventRepo.Update(ctx, event)
	if err != nil {
		return nil, err
	}
	s.publishPayEventInvalidation(ctx, updated, permission.OpUpdate, actor.UserID)
	if s.driverNotify != nil {
		s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: tenantInfo,
			WorkerID:   updated.WorkerID,
			EventType:  "dash.pay_held",
			Priority:   notification.PriorityHigh,
			Context: documenttemplate.SettlementNotificationContext{
				Reason: reason,
			},
			Link: "/dash",
			RelatedEntities: map[string]any{
				"payEventId": updated.ID.String(),
			},
		})
	}
	return updated, nil
}

func (s *Service) ReleasePayEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.PayEvent, error) {
	if err := requireActor(actor, "Pay event release"); err != nil {
		return nil, err
	}
	event, err := s.payEventRepo.GetByID(ctx, repositories.GetPayEventByIDRequest{
		ID:         eventID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !PlanReleasePayEvent(event) {
		return event, nil
	}

	updated, err := s.payEventRepo.Update(ctx, event)
	if err != nil {
		return nil, err
	}

	control, controlErr := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if controlErr == nil && control.AutoAttachAccruals {
		if refreshErr := s.refreshOpenDraftForWorker(
			ctx,
			tenantInfo,
			event.WorkerID,
		); refreshErr != nil {
			s.l.Error("failed to attach released pay event to open draft settlement",
				zap.Error(refreshErr))
		}
	}
	s.publishPayEventInvalidation(ctx, updated, permission.OpUpdate, actor.UserID)
	return updated, nil
}

func (s *Service) AttachPayEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	eventIDs []pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Pay event attachment"); err != nil {
		return nil, err
	}
	eventIDs = sliceutils.Dedupe(eventIDs)
	if len(eventIDs) == 0 {
		return nil, errNoPayEventsSelected()
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if txErr = checkDraftForEvents(entity, "added to"); txErr != nil {
			return txErr
		}
		previous = *entity

		for _, eventID := range eventIDs {
			txErr = s.appendPayEventToSettlement(txCtx, tenantInfo, entity, eventID)
			if txErr != nil {
				return txErr
			}
		}

		if txErr = s.settlementRepo.ReplaceLines(txCtx, entity); txErr != nil {
			return txErr
		}
		if txErr = s.payEventRepo.MarkSettled(
			txCtx,
			tenantInfo,
			eventIDs,
			entity.ID,
		); txErr != nil {
			return txErr
		}

		FinishAttach(entity)
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Pay events added to settlement")
	return updated, nil
}

func (s *Service) appendPayEventToSettlement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *driversettlement.Settlement,
	eventID pulid.ID,
) error {
	event, err := s.payEventRepo.GetByID(
		ctx,
		repositories.GetPayEventByIDRequest{ID: eventID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return err
	}
	if err = checkAttachable(entity, event); err != nil {
		return err
	}

	if PlanReleasePayEvent(event) {
		if _, err = s.payEventRepo.Update(ctx, event); err != nil {
			return err
		}
	}

	AppendPayEventLines(entity, event)
	return nil
}

func (s *Service) DetachPayEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID, eventID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Pay event removal"); err != nil {
		return nil, err
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if txErr = checkDraftForEvents(entity, "removed from"); txErr != nil {
			return txErr
		}
		previous = *entity

		event, txErr := s.payEventRepo.GetByID(
			txCtx,
			repositories.GetPayEventByIDRequest{ID: eventID, TenantInfo: tenantInfo},
		)
		if txErr != nil {
			return txErr
		}
		if txErr = PlanDetachPayEvent(entity, event); txErr != nil {
			return txErr
		}

		if txErr = s.settlementRepo.ReplaceLines(txCtx, entity); txErr != nil {
			return txErr
		}
		if txErr = s.payEventRepo.ReleaseEvents(
			txCtx,
			tenantInfo,
			[]pulid.ID{eventID},
		); txErr != nil {
			return txErr
		}
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpUpdate,
		"Pay event removed from settlement")
	return updated, nil
}

func countSettlementShipments(lines []*driversettlement.SettlementLine) int {
	shipments := make(map[pulid.ID]struct{}, len(lines))
	for _, line := range lines {
		if line != nil && line.ShipmentID != nil && !line.ShipmentID.IsNil() {
			shipments[*line.ShipmentID] = struct{}{}
		}
	}
	return len(shipments)
}
