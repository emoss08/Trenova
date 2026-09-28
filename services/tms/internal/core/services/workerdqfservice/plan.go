package workerdqfservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type VerificationChange = services.RecordChange[worker.WorkerEmploymentVerification]

func (s *Service) prepareRecord(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
	userID pulid.ID,
) error {
	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: verificationTenant(entity),
	}); err != nil {
		return err
	}

	entity.RequestedByID = userID
	return s.prepare(entity)
}

// PlanRecordVerification is the previous employer RecordVerification would
// file, without filing it.
func (s *Service) PlanRecordVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
	userID pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	planned := *entity
	if err := s.prepareRecord(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) loadVerification(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	return s.repo.GetVerificationByID(ctx, &repositories.GetEmploymentVerificationByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) PlanUpdateVerification(
	ctx context.Context,
	req *UpdateVerificationRequest,
) (*VerificationChange, error) {
	original, err := s.loadVerification(ctx, req.TenantInfo, req.VerificationID)
	if err != nil {
		return nil, err
	}
	entity := *original
	applyVerificationUpdate(&entity, req)
	if err = s.prepare(&entity); err != nil {
		return nil, err
	}
	return &VerificationChange{Before: original, After: &entity}, nil
}

// PlanMarkRequested is what MarkRequested would record: the request went out
// now.
func (s *Service) PlanMarkRequested(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*VerificationChange, error) {
	now := timeutils.NowUnix()
	status := worker.VerificationRequested
	return s.PlanUpdateVerification(ctx, &UpdateVerificationRequest{
		TenantInfo:     tenantInfo,
		VerificationID: id,
		Status:         &status,
		RequestedAt:    &now,
	})
}

func (s *Service) PlanRecordFollowUp(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*VerificationChange, error) {
	original, err := s.loadVerification(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	if original.Status != worker.VerificationRequested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only a request that is awaiting a response can be chased",
		)
	}

	entity := *original
	now := timeutils.NowUnix()
	entity.FollowUpCount++
	entity.LastFollowUpAt = &now
	if err = s.prepare(&entity); err != nil {
		return nil, err
	}
	return &VerificationChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanDeleteVerification(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	return s.loadVerification(ctx, tenantInfo, id)
}
