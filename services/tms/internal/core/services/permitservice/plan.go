package permitservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func validatePermit(entity *permit.Permit) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *service) PlanCreatePermit(
	_ context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	if err := validatePermit(entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *service) PlanUpdatePermit(
	ctx context.Context,
	entity *permit.Permit,
) (*services.RecordChange[permit.Permit], error) {
	if err := validatePermit(entity); err != nil {
		return nil, err
	}
	existing, err := s.permitRepo.GetByID(ctx, &repositories.GetPermitByIDRequest{
		PermitID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	if existing.ShipmentID != entity.ShipmentID {
		return nil, errortypes.NewValidationError(
			"permitId",
			errortypes.ErrInvalid,
			"This permit belongs to another shipment",
		)
	}
	return &services.RecordChange[permit.Permit]{Before: existing, After: entity}, nil
}
