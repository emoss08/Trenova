package recurringshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// SeriesChange is a recurring shipment before and after a change; Before is
// nil for one being made.
type SeriesChange struct {
	Before *recurringshipment.RecurringShipment
	After  *recurringshipment.RecurringShipment
}

func (s *Service) PreviewCreate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) (*SeriesChange, error) {
	if err := s.planCreate(ctx, entity, userID); err != nil {
		return nil, err
	}

	return &SeriesChange{After: entity}, nil
}

func (s *Service) planCreate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) error {
	if entity.Status == "" {
		entity.Status = recurringshipment.StatusActive
	}

	if entity.ExceptionPolicy == "" {
		entity.ExceptionPolicy = recurringshipment.ExceptionPolicySkip
	}

	entity.EnteredByID = userID

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return multiErr
	}

	return s.repo.Derive(ctx, entity)
}

func (s *Service) PreviewUpdate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*SeriesChange, error) {
	original, err := s.planUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}

	return &SeriesChange{Before: original, After: entity}, nil
}

func (s *Service) planUpdate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetRecurringShipmentByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		return nil, err
	}

	if err = s.repo.Derive(ctx, entity); err != nil {
		return nil, err
	}

	return original, nil
}

func (s *Service) PreviewUpdateStatus(
	ctx context.Context,
	req *repositories.UpdateRecurringShipmentStatusRequest,
) (*SeriesChange, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetRecurringShipmentByIDRequest{
		ID:         req.RecurringShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	after := *original
	after.Version = req.Version
	if err = after.ApplyStatus(req.Status, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return &SeriesChange{Before: original, After: &after}, nil
}

func (s *Service) PreviewGenerate(
	ctx context.Context,
	req *repositories.GenerateRecurringShipmentRequest,
) (*repositories.RecurringShipmentGenerationPlan, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if req.Trigger == "" {
		req.Trigger = recurringshipment.RunTriggerManual
	}

	return s.repo.PlanGenerate(ctx, req)
}
