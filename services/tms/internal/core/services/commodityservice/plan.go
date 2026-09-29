package commodityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/statuschange"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	if err := s.transformer.TransformCommodity(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *commodity.Commodity,
) (*services.RecordChange[commodity.Commodity], error) {
	if err := s.transformer.TransformCommodity(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCommodityByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &services.RecordChange[commodity.Commodity]{Before: original, After: entity}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCommodityStatusRequest,
) ([]services.RecordChange[commodity.Commodity], error) {
	found, err := s.repo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
		TenantInfo:   req.TenantInfo,
		CommodityIDs: req.CommodityIDs,
	})
	if err != nil {
		return nil, err
	}

	return statuschange.Plan(&statuschange.Request[commodity.Commodity, domaintypes.Status]{
		Field:  "commodityIds",
		Kind:   "commodity",
		IDs:    req.CommodityIDs,
		Found:  found,
		Status: req.Status,
		Valid:  req.Status.IsValid(),
		IDOf:   func(entity *commodity.Commodity) pulid.ID { return entity.ID },
		Set:    func(entity *commodity.Commodity, status domaintypes.Status) { entity.Status = status },
	})
}
