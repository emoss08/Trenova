package driversettlementservice

import (
	"context"

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
	"github.com/shopspring/decimal"
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
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A hold reason is required so the driver knows why pay was deferred",
		)
	}

	event, err := s.payEventRepo.GetByID(ctx, repositories.GetPayEventByIDRequest{
		ID:         eventID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if event.Status != driversettlement.PayEventStatusAccrued {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only accrued, unsettled pay events can be held",
		)
	}
	if event.OnHold {
		return event, nil
	}

	event.OnHold = true
	event.HoldReason = reason
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
			Title:      "Pay on hold",
			Message:    "Pay for one of your loads was placed on hold: " + reason,
			Link:       "/dash",
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
	if !event.OnHold {
		return event, nil
	}

	event.OnHold = false
	event.HoldReason = ""
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
	if len(eventIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrRequired,
			"Select at least one pay event to add",
		)
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
				"Pay events can only be added to draft settlements",
			)
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

		entity.ShipmentCount = countSettlementShipments(entity.Lines)
		entity.SyncTotals()
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
	if event.Status != driversettlement.PayEventStatusAccrued {
		return errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Pay event "+event.ProNumber+" is not accrued; only unsettled events can be added",
		)
	}
	if event.WorkerID != entity.WorkerID {
		return errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Pay event "+event.ProNumber+" belongs to a different driver",
		)
	}

	if event.OnHold {
		event.OnHold = false
		event.HoldReason = ""
		if _, err = s.payEventRepo.Update(ctx, event); err != nil {
			return err
		}
	}

	eventRef := event.ID
	shipmentID := event.ShipmentID
	for _, comp := range event.Components {
		entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
			Category:      driversettlement.LineCategoryEarning,
			ComponentKind: comp.Kind,
			Method:        comp.Method,
			Description:   comp.Description,
			Quantity:      comp.Quantity,
			Rate:          comp.Rate,
			AmountMinor:   comp.AmountMinor,
			ShipmentID:    &shipmentID,
			MoveID:        event.MoveID,
			PayEventID:    &eventRef,
			ProNumber:     event.ProNumber,
		})
	}
	entity.TotalMiles = entity.TotalMiles.Add(event.TotalMiles)
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
		if entity.Status != driversettlement.StatusDraft {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Pay events can only be removed from draft settlements",
			)
		}
		previous = *entity

		found := false
		remaining := make([]*driversettlement.SettlementLine, 0, len(entity.Lines))
		for _, line := range entity.Lines {
			if line == nil {
				continue
			}
			if line.PayEventID != nil && *line.PayEventID == eventID {
				found = true
				continue
			}
			remaining = append(remaining, line)
		}
		if !found {
			return errortypes.NewValidationError(
				"payEventId",
				errortypes.ErrInvalid,
				"Pay event is not part of this settlement",
			)
		}

		event, txErr := s.payEventRepo.GetByID(
			txCtx,
			repositories.GetPayEventByIDRequest{ID: eventID, TenantInfo: tenantInfo},
		)
		if txErr != nil {
			return txErr
		}

		entity.Lines = remaining
		entity.TotalMiles = entity.TotalMiles.Sub(event.TotalMiles)
		if entity.TotalMiles.IsNegative() {
			entity.TotalMiles = decimal.Zero
		}
		entity.ShipmentCount = countSettlementShipments(remaining)
		entity.SyncTotals()

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
