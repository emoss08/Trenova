package equipmentavailabilityhelper

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnsureAssignmentEquipmentAvailable(
	ctx context.Context,
	assignmentRepo repositories.AssignmentRepository,
	tenantInfo pagination.TenantInfo,
	assignment *shipment.Assignment,
	moveID pulid.ID,
) error {
	if assignment == nil || assignmentRepo == nil {
		return nil
	}

	if assignment.TractorID != nil {
		conflictingTractor, err := assignmentRepo.FindInProgressByTractorID(
			ctx,
			tenantInfo,
			*assignment.TractorID,
			moveID,
		)
		if err != nil {
			return err
		}
		if conflictingTractor != nil {
			return errortypes.NewBusinessError("Tractor is currently in progress on another move").
				WithParam("tractorId", assignment.TractorID.String()).
				WithParam("shipmentMoveId", conflictingTractor.ShipmentMoveID.String())
		}
	}

	if assignment.TrailerID == nil {
		return nil
	}

	conflictingTrailer, err := assignmentRepo.FindInProgressByTrailerID(
		ctx,
		tenantInfo,
		*assignment.TrailerID,
		moveID,
	)
	if err != nil {
		return err
	}
	if conflictingTrailer != nil {
		return errortypes.NewBusinessError("Trailer is currently in progress on another move").
			WithParam("trailerId", assignment.TrailerID.String()).
			WithParam("shipmentMoveId", conflictingTrailer.ShipmentMoveID.String())
	}

	return nil
}
