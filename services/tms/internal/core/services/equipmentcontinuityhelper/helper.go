package equipmentcontinuityhelper

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func AdvanceForCompletedMove(
	ctx context.Context,
	continuityRepo repositories.EquipmentContinuityRepository,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
) error {
	if continuityRepo == nil || move == nil || move.Assignment == nil || !move.IsCompleted() {
		return nil
	}

	destinationStop := lastDeliveryStop(move)
	if destinationStop == nil {
		return errortypes.NewBusinessError("Shipment move is missing a delivery stop").
			WithParam("shipmentMoveId", move.ID.String())
	}

	if move.Assignment.TractorID != nil {
		if _, err := continuityRepo.Advance(ctx, repositories.CreateEquipmentContinuityRequest{
			TenantInfo:           tenantInfo,
			EquipmentType:        equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:          *move.Assignment.TractorID,
			CurrentLocationID:    destinationStop.LocationID,
			SourceType:           equipmentcontinuity.SourceTypeAssignment,
			SourceShipmentID:     move.ShipmentID,
			SourceShipmentMoveID: move.ID,
			SourceAssignmentID:   move.Assignment.ID,
		}); err != nil {
			return err
		}
	}

	if move.Assignment.TrailerID == nil {
		return nil
	}

	_, err := continuityRepo.Advance(ctx, repositories.CreateEquipmentContinuityRequest{
		TenantInfo:           tenantInfo,
		EquipmentType:        equipmentcontinuity.EquipmentTypeTrailer,
		EquipmentID:          *move.Assignment.TrailerID,
		CurrentLocationID:    destinationStop.LocationID,
		SourceType:           equipmentcontinuity.SourceTypeAssignment,
		SourceShipmentID:     move.ShipmentID,
		SourceShipmentMoveID: move.ID,
		SourceAssignmentID:   move.Assignment.ID,
	})
	return err
}

func lastDeliveryStop(move *shipment.ShipmentMove) *shipment.Stop {
	var candidate *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || stop.Type != shipment.StopTypeDelivery {
			continue
		}
		if candidate == nil || stop.Sequence > candidate.Sequence {
			candidate = stop
		}
	}

	return candidate
}
