package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/app/statuschange"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *carrier.Carrier,
) (*kernelports.RecordChange[carrier.Carrier], error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeContacts:          true,
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &kernelports.RecordChange[carrier.Carrier]{Before: original, After: entity}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *carrierports.BulkUpdateCarrierStatusRequest,
) ([]kernelports.RecordChange[carrier.Carrier], error) {
	found, err := s.repo.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
		TenantInfo: req.TenantInfo,
		CarrierIDs: req.CarrierIDs,
	})
	if err != nil {
		return nil, err
	}

	return statuschange.Plan(&statuschange.Request[carrier.Carrier, carrier.Status]{
		Field:  "carrierIds",
		Kind:   "carrier",
		IDs:    req.CarrierIDs,
		Found:  found,
		Status: req.Status,
		Valid:  req.Status.IsValid(),
		IDOf:   func(entity *carrier.Carrier) pulid.ID { return entity.ID },
		Set:    func(entity *carrier.Carrier, status carrier.Status) { entity.Status = status },
	})
}
