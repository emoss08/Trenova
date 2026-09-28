package workerinjuryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type InjuryChange = services.RecordChange[worker.WorkerInjury]

func (s *Service) prepareRecord(
	ctx context.Context,
	entity *worker.WorkerInjury,
	userID pulid.ID,
) error {
	tenantInfo := injuryTenant(entity)
	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	entity.RecordedByID = userID
	if entity.CaseYear <= 0 {
		entity.CaseYear = yearOf(entity.OccurredAt)
	}
	if entity.CaseNumber <= 0 {
		next, err := s.repo.NextCaseNumber(ctx, &repositories.NextCaseNumberRequest{
			TenantInfo: tenantInfo,
			CaseYear:   entity.CaseYear,
		})
		if err != nil {
			return err
		}
		entity.CaseNumber = next
	}
	return s.prepare(entity)
}

func (s *Service) PlanRecordInjury(
	ctx context.Context,
	entity *worker.WorkerInjury,
	userID pulid.ID,
) (*worker.WorkerInjury, error) {
	planned := *entity
	if err := s.prepareRecord(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) PlanUpdateInjury(
	ctx context.Context,
	req *UpdateInjuryRequest,
) (*InjuryChange, error) {
	original, err := s.repo.GetInjuryByID(ctx, &repositories.GetWorkerInjuryByIDRequest{
		ID:         req.InjuryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	entity := *original
	ApplyInjuryUpdate(&entity, req)
	if err = s.prepare(&entity); err != nil {
		return nil, err
	}
	return &InjuryChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanDeleteInjury(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerInjury, error) {
	return s.repo.GetInjuryByID(ctx, &repositories.GetWorkerInjuryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}
