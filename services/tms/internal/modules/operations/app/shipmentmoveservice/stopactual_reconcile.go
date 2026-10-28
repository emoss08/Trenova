package shipmentmoveservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) ReconcileStopActuals(
	ctx context.Context,
	req *shipmentmoveports.ReconcileStopActualsRequest,
) (*shipmentmoveports.ReconcileStopActualsResult, error) {
	if multiErr := validateReconcileRequest(req); multiErr != nil {
		return nil, multiErr
	}

	result := &shipmentmoveports.ReconcileStopActualsResult{}
	var previousStatus shipment.MoveStatus
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		move, err := s.repo.GetByID(txCtx, &shipmentmoveports.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}
		previousStatus = move.Status

		touched, changes, err := settleStopActuals(txCtx, move, req.Plan)
		if err != nil {
			return err
		}
		result.Changes = changes
		if len(touched) == 0 {
			result.Move = move
			return nil
		}

		targetStatus := deriveMoveStatusFromStops(move)
		if err = s.checkReconciledStatus(
			txCtx,
			req,
			move,
			previousStatus,
			targetStatus,
		); err != nil {
			return err
		}

		for _, stop := range touched {
			if _, err = s.repo.UpdateStopActuals(txCtx, req.TenantInfo, stop); err != nil {
				return err
			}
		}

		result.Move, err = s.applyDerivedMoveStatus(
			txCtx,
			&shipmentmoveports.RecordStopActualRequest{
				TenantInfo: req.TenantInfo,
				MoveID:     req.MoveID,
			},
			previousStatus,
			targetStatus,
		)
		if err != nil {
			return err
		}

		return s.refreshShipmentState(txCtx, move.ShipmentID, req.TenantInfo)
	})
	if err != nil {
		return nil, err
	}

	if len(result.Changes) > 0 {
		s.announceStopActuals(
			ctx,
			req.TenantInfo,
			result.Move,
			previousStatus,
			recordedActions(req, result.Changes),
		)
	}

	return result, nil
}

func validateReconcileRequest(
	req *shipmentmoveports.ReconcileStopActualsRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	if req.TenantInfo.OrgID.IsNil() {
		multiErr.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if req.TenantInfo.BuID.IsNil() {
		multiErr.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if req.MoveID.IsNil() {
		multiErr.Add("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	if req.Plan == nil {
		multiErr.Add("plan", errortypes.ErrRequired, "A plan is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func settleStopActuals(
	ctx context.Context,
	move *shipment.ShipmentMove,
	plan shipmentmoveports.StopActualPlanner,
) ([]*shipment.Stop, []shipment.StopActualChange, error) {
	touched := make([]*shipment.Stop, 0)
	seen := make(map[pulid.ID]struct{})
	applied := make([]shipment.StopActualChange, 0)

	maxPasses := 2*len(move.Stops) + 2
	for range maxPasses {
		changes, err := plan(ctx, move)
		if err != nil {
			return nil, nil, err
		}
		if len(changes) == 0 {
			return touched, applied, nil
		}

		stops, err := move.ApplyStopActualChanges(changes)
		if err != nil {
			return nil, nil, err
		}
		for _, stop := range stops {
			if _, ok := seen[stop.ID]; !ok {
				seen[stop.ID] = struct{}{}
				touched = append(touched, stop)
			}
		}
		applied = append(applied, changes...)
	}

	return nil, nil, errortypes.NewBusinessError(
		"The stop times on this load did not settle",
	).WithParam("moveId", move.ID.String())
}

func (s *service) checkReconciledStatus(
	ctx context.Context,
	req *shipmentmoveports.ReconcileStopActualsRequest,
	move *shipment.ShipmentMove,
	previousStatus, targetStatus shipment.MoveStatus,
) error {
	if targetStatus == previousStatus {
		return nil
	}
	if targetStatus == shipment.MoveStatusInTransit {
		if err := s.ensureEquipmentAvailableForProgress(ctx, req.TenantInfo, move.ID); err != nil {
			return err
		}
	}
	if err := s.ensureNoDeliveryHold(
		ctx,
		move.ShipmentID,
		req.TenantInfo,
		targetStatus,
	); err != nil {
		return err
	}
	if !shipmentstate.CanTransitionMoveStatus(previousStatus, targetStatus) {
		return errortypes.NewBusinessError(
			"Move status transition from {0} to {1} is not allowed",
			previousStatus,
			targetStatus,
		).WithParam("moveId", req.MoveID.String())
	}
	return nil
}

func recordedActions(
	req *shipmentmoveports.ReconcileStopActualsRequest,
	changes []shipment.StopActualChange,
) []*shipmentmoveports.RecordStopActualRequest {
	actions := make([]*shipmentmoveports.RecordStopActualRequest, 0, 2*len(changes))
	for _, change := range changes {
		if change.Arrival != nil {
			actions = append(actions, &shipmentmoveports.RecordStopActualRequest{
				TenantInfo: req.TenantInfo,
				MoveID:     req.MoveID,
				StopID:     change.StopID,
				Action:     shipmentmoveports.StopActualActionArrive,
				OccurredAt: change.Arrival,
			})
		}
		if change.Departure != nil {
			actions = append(actions, &shipmentmoveports.RecordStopActualRequest{
				TenantInfo: req.TenantInfo,
				MoveID:     req.MoveID,
				StopID:     change.StopID,
				Action:     shipmentmoveports.StopActualActionDepart,
				OccurredAt: change.Departure,
			})
		}
	}
	return actions
}
