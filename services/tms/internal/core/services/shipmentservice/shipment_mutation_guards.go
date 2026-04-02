package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/equipmentavailabilityhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) ensureEquipmentAvailableForShipmentUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
) error {
	if s.assignmentRepo == nil || updated == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}

	originalMoveCap := 0
	if original != nil {
		originalMoveCap = len(original.Moves)
	}
	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, originalMoveCap)
	if original != nil {
		for _, move := range original.Moves {
			if move == nil || move.ID.IsNil() {
				continue
			}
			originalMoves[move.ID] = move
		}
	}

	seenTractors := make(map[pulid.ID]pulid.ID)
	seenTrailers := make(map[pulid.ID]pulid.ID)

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || !move.IsInTransit() {
			continue
		}

		previous := originalMoves[move.ID]
		if previous != nil && previous.IsInTransit() {
			continue
		}
		if move.Assignment == nil {
			continue
		}

		if move.Assignment.TractorID != nil {
			if priorMoveID, ok := seenTractors[*move.Assignment.TractorID]; ok {
				return errortypes.NewBusinessError("Tractor is currently in progress on another move").
					WithParam("tractorId", move.Assignment.TractorID.String()).
					WithParam("shipmentMoveId", priorMoveID.String())
			}
			seenTractors[*move.Assignment.TractorID] = move.ID
		}

		if move.Assignment.TrailerID != nil {
			if priorMoveID, ok := seenTrailers[*move.Assignment.TrailerID]; ok {
				return errortypes.NewBusinessError("Trailer is currently in progress on another move").
					WithParam("trailerId", move.Assignment.TrailerID.String()).
					WithParam("shipmentMoveId", priorMoveID.String())
			}
			seenTrailers[*move.Assignment.TrailerID] = move.ID
		}

		if err := equipmentavailabilityhelper.EnsureAssignmentEquipmentAvailable(
			ctx,
			s.assignmentRepo,
			tenantInfo,
			move.Assignment,
			move.ID,
		); err != nil {
			return err
		}
	}

	return nil
}
