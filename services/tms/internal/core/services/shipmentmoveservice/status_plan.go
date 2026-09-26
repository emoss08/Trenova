package shipmentmoveservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type moveStatusRequest struct {
	tenantInfo pagination.TenantInfo
	moveIDs    []pulid.ID
	status     shipment.MoveStatus
	// forUpdate locks each move for the write; a preview reads without it.
	forUpdate bool
}

// moveStatusPlan is everything a status change decides before it writes:
// each move as it stands, the status it held, and the shipments to re-derive
// in the order their moves were asked for.
type moveStatusPlan struct {
	moves       []*shipment.ShipmentMove
	previous    map[pulid.ID]shipment.MoveStatus
	shipmentIDs []pulid.ID
}

// PreviewUpdateStatus is what UpdateStatus or BulkUpdateStatus would do, from
// the same checks, without writing: each move with its new status and each
// shipment with its state re-derived by the coordinator.
func (s *service) PreviewUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateMoveStatusRequest,
) (*services.MoveStatusPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if err := validateUniqueMoveIDs(req.MoveIDs); err != nil {
		return nil, err
	}

	plan, err := s.planMoveStatus(ctx, &moveStatusRequest{
		tenantInfo: req.TenantInfo,
		moveIDs:    req.MoveIDs,
		status:     req.Status,
	})
	if err != nil {
		return nil, err
	}

	out := &services.MoveStatusPlan{
		MovesBefore:     plan.moves,
		MovesAfter:      make([]*shipment.ShipmentMove, 0, len(plan.moves)),
		ShipmentsBefore: make([]*shipment.Shipment, 0, len(plan.shipmentIDs)),
		ShipmentsAfter:  make([]*shipment.Shipment, 0, len(plan.shipmentIDs)),
	}
	for _, move := range plan.moves {
		after := new(shipment.ShipmentMove)
		if err = jsonutils.Convert(move, after); err != nil {
			return nil, err
		}
		after.Status = req.Status
		out.MovesAfter = append(out.MovesAfter, after)
	}

	for _, shipmentID := range plan.shipmentIDs {
		before, after, pErr := s.projectShipmentState(ctx, shipmentID, req)
		if pErr != nil {
			return nil, pErr
		}
		out.ShipmentsBefore = append(out.ShipmentsBefore, before)
		out.ShipmentsAfter = append(out.ShipmentsAfter, after)
	}

	return out, nil
}

// planMoveStatus checks every move the way the write does: the transition is
// allowed, a move going in transit has its equipment free (and no two moves
// in one request share a tractor or trailer), and nothing completes under a
// delivery-blocking hold.
func (s *service) planMoveStatus(
	ctx context.Context,
	req *moveStatusRequest,
) (*moveStatusPlan, error) {
	plan := &moveStatusPlan{
		moves:       make([]*shipment.ShipmentMove, 0, len(req.moveIDs)),
		previous:    make(map[pulid.ID]shipment.MoveStatus, len(req.moveIDs)),
		shipmentIDs: make([]pulid.ID, 0, len(req.moveIDs)),
	}
	seenTractors := make(map[pulid.ID]pulid.ID, len(req.moveIDs))
	seenTrailers := make(map[pulid.ID]pulid.ID, len(req.moveIDs))
	seenShipments := make(map[pulid.ID]struct{}, len(req.moveIDs))

	for _, moveID := range req.moveIDs {
		move, err := s.repo.GetByID(ctx, &repositories.GetMoveByIDRequest{
			MoveID:            moveID,
			TenantInfo:        req.tenantInfo,
			ExpandMoveDetails: false,
			ForUpdate:         req.forUpdate,
		})
		if err != nil {
			return nil, err
		}
		plan.previous[moveID] = move.Status

		if !shipmentstate.CanTransitionMoveStatus(move.Status, req.status) {
			return nil, errortypes.NewBusinessError(
				"Move status transition from {0} to {1} is not allowed",
				move.Status,
				req.status,
			).WithParam("moveId", moveID.String())
		}
		if err = s.ensureEquipmentAvailableForProgressBulk(
			ctx,
			req.tenantInfo,
			move.ID,
			req.status,
			seenTractors,
			seenTrailers,
		); err != nil {
			return nil, err
		}
		if err = s.ensureNoDeliveryHold(
			ctx,
			move.ShipmentID,
			req.tenantInfo,
			req.status,
		); err != nil {
			return nil, err
		}

		plan.moves = append(plan.moves, move)
		if _, seen := seenShipments[move.ShipmentID]; !seen {
			seenShipments[move.ShipmentID] = struct{}{}
			plan.shipmentIDs = append(plan.shipmentIDs, move.ShipmentID)
		}
	}

	return plan, nil
}

// projectShipmentState is the shipment as stored and as refreshShipmentState
// would re-derive it once the moves carry their new status.
func (s *service) projectShipmentState(
	ctx context.Context,
	shipmentID pulid.ID,
	req *repositories.BulkUpdateMoveStatusRequest,
) (before, after *shipment.Shipment, err error) {
	before, err = s.loadShipment(ctx, shipmentID, req.TenantInfo)
	if err != nil {
		return nil, nil, err
	}

	after = shipment.CloneForUpdate(before)
	for _, moveID := range req.MoveIDs {
		if move := after.FindMove(moveID); move != nil {
			move.Status = req.Status
		}
	}

	if _, err = s.deriveShipmentState(ctx, after, req.TenantInfo); err != nil {
		return nil, nil, err
	}

	return before, after, nil
}
