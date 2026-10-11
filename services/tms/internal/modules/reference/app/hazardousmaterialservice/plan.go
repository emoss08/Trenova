package hazardousmaterialservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/statuschange"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	if err := s.transformer.TransformHazardousMaterial(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
	if err := s.transformer.TransformHazardousMaterial(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, hazardousmaterialports.GetHazardousMaterialByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &kernelports.RecordChange[hazardousmaterial.HazardousMaterial]{
		Before: original,
		After:  entity,
	}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
) ([]kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
	found, err := s.repo.GetByIDs(ctx, hazardousmaterialports.GetHazardousMaterialsByIDsRequest{
		TenantInfo:           req.TenantInfo,
		HazardousMaterialIDs: req.HazardousMaterialIDs,
	})
	if err != nil {
		return nil, err
	}

	return statuschange.Plan(
		&statuschange.Request[hazardousmaterial.HazardousMaterial, domaintypes.Status]{
			Field:  "hazardousMaterialIds",
			Kind:   "hazardous material",
			IDs:    req.HazardousMaterialIDs,
			Found:  found,
			Status: req.Status,
			Valid:  req.Status.IsValid(),
			IDOf:   func(entity *hazardousmaterial.HazardousMaterial) pulid.ID { return entity.ID },
			Set:    func(entity *hazardousmaterial.HazardousMaterial, status domaintypes.Status) { entity.Status = status },
		},
	)
}
