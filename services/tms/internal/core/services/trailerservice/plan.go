package trailerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LocatePlan struct {
	Trailer  *trailer.Trailer
	Location *location.Location
	Current  *equipmentcontinuity.EquipmentContinuity
	Previous *shipment.Shipment
	Updated  *shipment.Shipment
}

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *trailer.Trailer,
) (*services.RecordChange[trailer.Trailer], error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetTrailerByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &services.RecordChange[trailer.Trailer]{Before: original, After: entity}, nil
}

func (s *Service) PlanLocate(
	ctx context.Context,
	req *repositories.LocateTrailerRequest,
	actor *services.RequestActor,
) (*LocatePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	plan, err := s.planLocate(ctx, req, actor.AuditActor().UserID)
	if err != nil {
		return nil, err
	}

	plan.Location, err = s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID:         req.NewLocationID,
		TenantInfo: pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID},
	})
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) planLocate(
	ctx context.Context,
	req *repositories.LocateTrailerRequest,
	actorUserID pulid.ID,
) (*LocatePlan, error) {
	trailerEntity, err := s.repo.GetByID(ctx, repositories.GetTrailerByIDRequest{
		ID: req.TrailerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}
	inProgress, err := s.assignmentRepo.FindInProgressByTrailerID(
		ctx,
		req.TenantInfo,
		req.TrailerID,
		pulid.Nil,
	)
	if err != nil {
		return nil, err
	}
	if inProgress != nil {
		return nil, errortypes.NewBusinessError("Trailer is currently in progress on another move").
			WithParam("trailerId", req.TrailerID.String()).
			WithParam("shipmentMoveId", inProgress.ShipmentMoveID.String())
	}

	current, err := s.continuityRepo.GetEffectiveCurrent(
		ctx,
		repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    req.TenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:   req.TrailerID,
		},
	)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, errortypes.NewBusinessError(
			"Trailer has no continuity history and does not need manual locate before dispatch",
		).WithParam("trailerId", req.TrailerID.String())
	}
	if current.SourceShipmentID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"Trailer continuity is missing the previous shipment association required for locate",
		).WithParam("trailerId", req.TrailerID.String())
	}
	if current.CurrentLocationID == req.NewLocationID {
		return nil, errortypes.NewBusinessError("Trailer is already located at the requested location").
			WithParam("trailerId", req.TrailerID.String())
	}

	previousShipment, updatedShipment, err := s.planLocateMove(ctx, req, current, actorUserID)
	if err != nil {
		return nil, err
	}

	return &LocatePlan{
		Trailer:  trailerEntity,
		Current:  current,
		Previous: previousShipment,
		Updated:  updatedShipment,
	}, nil
}

func (s *Service) planLocateMove(
	ctx context.Context,
	req *repositories.LocateTrailerRequest,
	current *equipmentcontinuity.EquipmentContinuity,
	actorUserID pulid.ID,
) (previous, updated *shipment.Shipment, err error) {
	previousShipment, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: current.SourceShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	updatedShipment := cloneShipment(previousShipment)
	timing := locateMoveTiming()
	appendLocateMove(updatedShipment, current, req, timing)

	control, err := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		previousShipment,
		updatedShipment,
		shipmentstate.ResolveControlDelayThreshold(control),
	); multiErr != nil {
		return nil, nil, multiErr
	}

	if err = s.commercial.Recalculate(
		ctx,
		updatedShipment,
		control,
		actorUserID,
	); err != nil {
		return nil, nil, err
	}
	if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(
		ctx,
		previousShipment,
		updatedShipment,
	); multiErr != nil {
		return nil, nil, multiErr
	}

	return previousShipment, updatedShipment, nil
}
