package customerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/statuschange"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *customer.Customer,
) (*customer.Customer, error) {
	if err := s.transformer.TransformCustomer(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *customer.Customer,
) (*services.RecordChange[customer.Customer], error) {
	if err := s.transformer.TransformCustomer(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &services.RecordChange[customer.Customer]{Before: original, After: entity}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCustomerStatusRequest,
) ([]services.RecordChange[customer.Customer], error) {
	found, err := s.repo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:  req.TenantInfo,
		CustomerIDs: req.CustomerIDs,
	})
	if err != nil {
		return nil, err
	}

	return statuschange.Plan(&statuschange.Request[customer.Customer, domaintypes.Status]{
		Field:  "customerIds",
		Kind:   "customer",
		IDs:    req.CustomerIDs,
		Found:  found,
		Status: req.Status,
		Valid:  req.Status.IsValid(),
		IDOf:   func(entity *customer.Customer) pulid.ID { return entity.ID },
		Set:    func(entity *customer.Customer, status domaintypes.Status) { entity.Status = status },
	})
}
