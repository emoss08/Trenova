package shipmentmoveservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *service) RecordStopActual(
	ctx context.Context,
	req *repositories.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var updatedMove *shipment.ShipmentMove
	var previousStatus shipment.MoveStatus
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, err := s.repo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}
		previousStatus = move.Status

		stop, err := applyStopActual(move, req)
		if err != nil {
			return err
		}

		targetStatus := deriveMoveStatusFromStops(move)
		if targetStatus == shipment.MoveStatusInTransit && previousStatus != targetStatus {
			if err = s.ensureEquipmentAvailableForProgress(
				txCtx,
				req.TenantInfo,
				move.ID,
			); err != nil {
				return err
			}
		}
		if err = s.ensureNoDeliveryHold(
			txCtx,
			move.ShipmentID,
			req.TenantInfo,
			targetStatus,
		); err != nil {
			return err
		}

		if _, err = s.repo.UpdateStopActuals(txCtx, req.TenantInfo, stop); err != nil {
			return err
		}

		updatedMove, err = s.applyDerivedMoveStatus(txCtx, req, previousStatus, targetStatus)
		if err != nil {
			return err
		}

		return s.refreshShipmentState(txCtx, move.ShipmentID, req.TenantInfo)
	})
	if err != nil {
		return nil, err
	}

	if updatedMove != nil && previousStatus != updatedMove.Status {
		s.recordMoveEvent(ctx, shipmenteventservice.BuildMoveStatusChanged(
			tenantRefForMoveTenant(req.TenantInfo),
			updatedMove,
			previousStatus,
			actorForMoveTenant(req.TenantInfo),
		))
		s.evaluateServiceFailuresAfterMoveStatus(ctx, updatedMove.ShipmentID, req.TenantInfo)
		s.notifyMoveObservers(ctx, req.TenantInfo, updatedMove, previousStatus)
	}
	if req.Action == repositories.StopActualActionDepart && updatedMove != nil {
		s.flagDetentionCandidate(ctx, req, updatedMove)
	}

	return updatedMove, nil
}

// defaultDetentionAlertThresholdMinutes is the fallback dwell time beyond
// which a stop is surfaced to dispatch as a detention-billing candidate when
// no DashControl is available. Driver detention PAY is computed separately
// from the pay profile's Detention component free time; this alert exists so
// the carrier can bill the customer accessorial. The per-org threshold and an
// on/off switch live on DashControl.
const defaultDetentionAlertThresholdMinutes = int64(120)

func (s *service) detentionAlertThreshold(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, bool) {
	if s.dashControlRepo == nil {
		return defaultDetentionAlertThresholdMinutes, true
	}
	control, err := s.dashControlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("failed to load dash control for detention alert", zap.Error(err))
		return defaultDetentionAlertThresholdMinutes, true
	}
	return int64(control.DetentionAlertThresholdMinutes), control.EnableDetentionAlerts
}

func (s *service) flagDetentionCandidate(
	ctx context.Context,
	req *repositories.RecordStopActualRequest,
	move *shipment.ShipmentMove,
) {
	if s.notifications == nil {
		return
	}
	thresholdMinutes, enabled := s.detentionAlertThreshold(ctx, req.TenantInfo)
	if !enabled {
		return
	}
	var stop *shipment.Stop
	for _, candidate := range move.Stops {
		if candidate != nil && candidate.ID == req.StopID {
			stop = candidate
			break
		}
	}
	if stop == nil || stop.ActualArrival == nil || stop.ActualDeparture == nil {
		return
	}

	dwellMinutes := (*stop.ActualDeparture - *stop.ActualArrival) / 60
	if dwellMinutes < thresholdMinutes {
		return
	}
	if stop.CountDetentionOverride != nil && !*stop.CountDetentionOverride {
		return
	}

	location := stop.AddressLine
	if stop.Location != nil && stop.Location.Name != "" {
		location = stop.Location.Name
	}
	buID := req.TenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      "detention_candidate",
		Priority:       notification.PriorityHigh,
		Channel:        notification.ChannelGlobal,
		Title:          "Detention candidate",
		Message: fmt.Sprintf(
			"Driver spent %d min at %s — beyond the %d-minute threshold. Review for detention billing.",
			dwellMinutes,
			location,
			thresholdMinutes,
		),
		Data: map[string]any{"link": "/shipment-management/shipments"},
		RelatedEntities: map[string]any{
			"shipmentId": move.ShipmentID.String(),
			"moveId":     move.ID.String(),
			"stopId":     stop.ID.String(),
		},
		Source: "shipment_move",
	}
	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create detention candidate notification", zap.Error(err))
	}
}

func (s *service) applyDerivedMoveStatus(
	ctx context.Context,
	req *repositories.RecordStopActualRequest,
	previousStatus, targetStatus shipment.MoveStatus,
) (*shipment.ShipmentMove, error) {
	if targetStatus == previousStatus {
		return s.repo.GetByID(ctx, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
		})
	}

	if !shipmentstate.CanTransitionMoveStatus(previousStatus, targetStatus) {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf(
				"Move status transition from %s to %s is not allowed",
				previousStatus,
				targetStatus,
			),
		).WithParam("moveId", req.MoveID.String())
	}

	updatedMove, err := s.repo.UpdateStatus(ctx, &repositories.UpdateMoveStatusRequest{
		TenantInfo: req.TenantInfo,
		MoveID:     req.MoveID,
		Status:     targetStatus,
	})
	if err != nil {
		return nil, err
	}
	if err = s.advanceEquipmentContinuityForMove(
		ctx,
		req.TenantInfo,
		updatedMove,
		targetStatus,
	); err != nil {
		return nil, err
	}
	return updatedMove, nil
}

func applyStopActual(
	move *shipment.ShipmentMove,
	req *repositories.RecordStopActualRequest,
) (*shipment.Stop, error) {
	if move.Status == shipment.MoveStatusCanceled {
		return nil, errortypes.NewBusinessError("This load has been canceled")
	}
	if move.Status == shipment.MoveStatusCompleted {
		return nil, errortypes.NewBusinessError("This load is already completed")
	}

	stops := activeStopsBySequence(move)
	var stop *shipment.Stop
	for _, candidate := range stops {
		if candidate.ID == req.StopID {
			stop = candidate
			break
		}
	}
	if stop == nil {
		return nil, errortypes.NewNotFoundError("Stop not found on this load")
	}

	now := timeutils.NowUnix()
	switch req.Action {
	case repositories.StopActualActionArrive:
		if stop.ActualArrival != nil {
			return nil, errortypes.NewBusinessError("You've already arrived at this stop")
		}
		for _, prior := range stops {
			if prior.Sequence >= stop.Sequence {
				break
			}
			if prior.ActualDeparture == nil {
				return nil, errortypes.NewBusinessError(
					"Complete the earlier stops on this load first",
				)
			}
		}
		stop.ActualArrival = &now
		stop.Status = shipment.StopStatusInTransit
	case repositories.StopActualActionDepart:
		if stop.ActualArrival == nil {
			return nil, errortypes.NewBusinessError("Arrive at this stop before departing")
		}
		if stop.ActualDeparture != nil {
			return nil, errortypes.NewBusinessError("You've already departed this stop")
		}
		stop.ActualDeparture = &now
		stop.Status = shipment.StopStatusCompleted
	default:
		return nil, errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Action must be Arrive or Depart",
		)
	}

	return stop, nil
}

func activeStopsBySequence(move *shipment.ShipmentMove) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(move.Stops))
	for _, stop := range move.Stops {
		if stop != nil && stop.Status != shipment.StopStatusCanceled {
			stops = append(stops, stop)
		}
	}
	sort.SliceStable(stops, func(i, j int) bool { return stops[i].Sequence < stops[j].Sequence })
	return stops
}

func deriveMoveStatusFromStops(move *shipment.ShipmentMove) shipment.MoveStatus {
	stops := activeStopsBySequence(move)
	if len(stops) == 0 {
		return move.Status
	}

	allCompleted := true
	anyProgress := false
	for _, stop := range stops {
		if stop.ActualArrival != nil || stop.ActualDeparture != nil {
			anyProgress = true
		}
		if stop.ActualArrival == nil || stop.ActualDeparture == nil {
			allCompleted = false
		}
	}

	switch {
	case allCompleted:
		return shipment.MoveStatusCompleted
	case anyProgress:
		return shipment.MoveStatusInTransit
	default:
		return move.Status
	}
}
