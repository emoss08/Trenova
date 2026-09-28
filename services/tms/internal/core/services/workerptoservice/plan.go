package workerptoservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PTOChange = services.RecordChange[worker.WorkerPTO]

func (s *Service) requireNoOverlap(
	ctx context.Context,
	entity *worker.WorkerPTO,
	excludeID pulid.ID,
) error {
	overlaps, err := s.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		WorkerID:  entity.WorkerID,
		StartDate: entity.StartDate,
		EndDate:   entity.EndDate,
		ExcludeID: excludeID,
	})
	if err != nil {
		return err
	}
	if overlaps {
		return errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"This worker already has pending or approved time off that overlaps these dates",
		)
	}
	return nil
}

func (s *Service) prepareCreate(
	ctx context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (bool, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return false, multiErr
	}
	if err := s.requireNoOverlap(ctx, entity, pulid.Nil); err != nil {
		return false, err
	}

	availability, err := s.prepareLedgerFields(ctx, entity, pulid.Nil)
	if err != nil {
		return false, err
	}

	autoApprove := availability != nil && availability.Policy != nil &&
		!availability.Policy.Policy.RequiresApproval
	if autoApprove {
		entity.Status = worker.PTOStatusApproved
		entity.ApproverID = userID
		entity.AutoApproved = true
	}
	return autoApprove, nil
}

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	planned := *entity
	if _, err := s.prepareCreate(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) prepareUpdate(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	current, err := s.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	if current.Status != worker.PTOStatusRequested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"PTO is {0} and can no longer be edited", strings.ToLower(string(current.Status)),
		)
	}

	entity.WorkerID = current.WorkerID
	entity.Status = current.Status
	entity.ApproverID = current.ApproverID
	entity.RejectorID = current.RejectorID
	entity.CancelledByID = current.CancelledByID
	entity.CreatedAt = current.CreatedAt

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err = s.requireNoOverlap(ctx, entity, entity.ID); err != nil {
		return nil, err
	}
	if _, err = s.prepareLedgerFields(ctx, entity, entity.ID); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *Service) PlanUpdate(ctx context.Context, entity *worker.WorkerPTO) (*PTOChange, error) {
	planned := *entity
	current, err := s.prepareUpdate(ctx, &planned)
	if err != nil {
		return nil, err
	}
	return &PTOChange{Before: current, After: &planned}, nil
}
