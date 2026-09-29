package tractorservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LocatePlan struct {
	Tractor  *tractor.Tractor
	Location *location.Location
	Current  *equipmentcontinuity.EquipmentContinuity
}

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *tractor.Tractor,
) (*services.RecordChange[tractor.Tractor], error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetTractorByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &services.RecordChange[tractor.Tractor]{Before: original, After: entity}, nil
}

func (s *Service) PlanLocate(
	ctx context.Context,
	req *repositories.LocateTractorRequest,
) (*LocatePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.planLocate(ctx, req)
}

func (s *Service) planLocate(
	ctx context.Context,
	req *repositories.LocateTractorRequest,
) (*LocatePlan, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetTractorByIDRequest{
		ID:         req.TractorID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	target, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID:         req.NewLocationID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	inProgress, err := s.assignmentRepo.FindInProgressByTractorID(
		ctx,
		req.TenantInfo,
		req.TractorID,
		pulid.Nil,
	)
	if err != nil {
		return nil, err
	}
	if inProgress != nil {
		return nil, errortypes.NewBusinessError("Tractor is currently in progress on another move").
			WithParam("tractorId", req.TractorID.String()).
			WithParam("shipmentMoveId", inProgress.ShipmentMoveID.String())
	}

	current, err := s.continuityRepo.GetEffectiveCurrent(
		ctx,
		repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    req.TenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   req.TractorID,
		},
	)
	if err != nil {
		return nil, err
	}
	if current != nil && current.CurrentLocationID == req.NewLocationID {
		return nil, errortypes.NewBusinessError(
			"Tractor is already located at the requested location",
		).WithParam("tractorId", req.TractorID.String())
	}

	return &LocatePlan{Tractor: entity, Location: target, Current: current}, nil
}
