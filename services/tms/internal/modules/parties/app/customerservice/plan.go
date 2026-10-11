package customerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/app/statuschange"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
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
) (*kernelports.RecordChange[customer.Customer], error) {
	if err := s.transformer.TransformCustomer(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &kernelports.RecordChange[customer.Customer]{Before: original, After: entity}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]kernelports.RecordChange[customer.Customer], error) {
	found, err := s.repo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
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
