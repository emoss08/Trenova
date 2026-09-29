package locationservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/statuschange"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *location.Location,
) (*services.RecordChange[location.Location], error) {
	if err := s.transformer.TransformLocation(ctx, entity); err != nil {
		return nil, err
	}
	entity.NormalizeGeofence()

	original, err := s.repo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(entity.Code) == "" {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("code", errortypes.ErrRequired, "Code is required")
		return nil, multiErr
	}
	if !strings.EqualFold(strings.TrimSpace(entity.Code), original.Code) {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("code", errortypes.ErrInvalid, "Location code cannot be changed")
		return nil, multiErr
	}
	entity.Code = original.Code
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return &services.RecordChange[location.Location]{Before: original, After: entity}, nil
}

func (s *Service) PlanBulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateLocationStatusRequest,
) ([]services.RecordChange[location.Location], error) {
	found, err := s.repo.GetByIDs(ctx, repositories.GetLocationsByIDsRequest{
		TenantInfo:  req.TenantInfo,
		LocationIDs: req.LocationIDs,
	})
	if err != nil {
		return nil, err
	}

	return statuschange.Plan(&statuschange.Request[location.Location, domaintypes.Status]{
		Field:  "locationIds",
		Kind:   "location",
		IDs:    req.LocationIDs,
		Found:  found,
		Status: req.Status,
		Valid:  req.Status.IsValid(),
		IDOf:   func(entity *location.Location) pulid.ID { return entity.ID },
		Set: func(entity *location.Location, status domaintypes.Status) {
			entity.Status = status
		},
	})
}
