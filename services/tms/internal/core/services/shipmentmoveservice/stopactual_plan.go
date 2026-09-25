package shipmentmoveservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
)

type stopActualPlan struct {
	stop           *shipment.Stop
	previousStatus shipment.MoveStatus
	targetStatus   shipment.MoveStatus
}

func (s *service) PreviewStopActual(
	ctx context.Context,
	req *repositories.RecordStopActualRequest,
) (*services.StopActualPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, err := s.repo.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            req.MoveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	before := new(shipment.ShipmentMove)
	if err = jsonutils.Convert(move, before); err != nil {
		return nil, err
	}

	plan, err := s.planStopActual(ctx, req, move)
	if err != nil {
		return nil, err
	}
	move.Status = plan.targetStatus

	return &services.StopActualPlan{Before: before, After: move}, nil
}

func (s *service) planStopActual(
	ctx context.Context,
	req *repositories.RecordStopActualRequest,
	move *shipment.ShipmentMove,
) (*stopActualPlan, error) {
	plan := &stopActualPlan{previousStatus: move.Status}

	stop, err := applyStopActual(move, req)
	if err != nil {
		return nil, err
	}
	plan.stop = stop

	plan.targetStatus = deriveMoveStatusFromStops(move)
	if plan.targetStatus == shipment.MoveStatusInTransit &&
		plan.previousStatus != plan.targetStatus {
		if err = s.ensureEquipmentAvailableForProgress(
			ctx,
			req.TenantInfo,
			move.ID,
		); err != nil {
			return nil, err
		}
	}
	if err = s.ensureNoDeliveryHold(
		ctx,
		move.ShipmentID,
		req.TenantInfo,
		plan.targetStatus,
	); err != nil {
		return nil, err
	}
	if plan.targetStatus != plan.previousStatus &&
		!shipmentstate.CanTransitionMoveStatus(plan.previousStatus, plan.targetStatus) {
		return nil, errortypes.NewBusinessError(
			"Move status transition from {0} to {1} is not allowed",
			plan.previousStatus,
			plan.targetStatus,
		).WithParam("moveId", req.MoveID.String())
	}

	return plan, nil
}
