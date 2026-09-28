package workerleaveservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type (
	CaseChange  = services.RecordChange[worker.WorkerLeaveCase]
	EntryChange = services.RecordChange[worker.WorkerLeaveEntry]
)

func (s *Service) prepareOpenCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
	userID pulid.ID,
) error {
	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: caseTenant(entity),
	}); err != nil {
		return err
	}

	entity.RecordedByID = userID
	return s.prepareCase(entity)
}

func (s *Service) PlanOpenCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
	userID pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	planned := *entity
	if err := s.prepareOpenCase(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) loadCase(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (original, leaveCase *worker.WorkerLeaveCase, err error) {
	original, err = s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	entity := *original
	return original, &entity, nil
}

func (s *Service) PlanUpdateCase(ctx context.Context, req *UpdateCaseRequest) (*CaseChange, error) {
	original, entity, err := s.loadCase(ctx, req.TenantInfo, req.CaseID)
	if err != nil {
		return nil, err
	}
	applyCaseUpdate(entity, req)
	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}
	return &CaseChange{Before: original, After: entity}, nil
}

func (s *Service) PlanCloseCase(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*CaseChange, error) {
	original, entity, err := s.loadCase(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	if original.Status == worker.LeaveCaseClosed {
		return &CaseChange{Before: original, After: entity}, nil
	}
	if original.Status == worker.LeaveCasePending {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Decide the case before closing it",
		)
	}

	now := timeutils.NowUnix()
	entity.Status = worker.LeaveCaseClosed
	entity.ClosedAt = &now
	if entity.EndsAt == nil {
		entity.EndsAt = &now
	}
	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}
	return &CaseChange{Before: original, After: entity}, nil
}

func (s *Service) PlanRequestCertification(
	ctx context.Context,
	req *RequestCertificationRequest,
) (*CaseChange, error) {
	original, entity, err := s.loadCase(ctx, req.TenantInfo, req.CaseID)
	if err != nil {
		return nil, err
	}

	control, err := s.repo.GetControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	due := now + int64(control.CertificationDueDays)*secondsPerDay
	if req.DueAt != nil && *req.DueAt > 0 {
		due = *req.DueAt
	}

	entity.CertificationStatus = worker.CertificationRequested
	entity.CertificationRequestedAt = &now
	entity.CertificationDueAt = &due
	entity.CertificationReceivedAt = nil

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}
	return &CaseChange{Before: original, After: entity}, nil
}

func (s *Service) PlanRecordDay(
	ctx context.Context,
	req *RecordDayRequest,
) (*worker.WorkerLeaveEntry, error) {
	leaveCase, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !leaveCase.IsOpen() {
		return nil, errortypes.NewValidationError(
			"leaveCaseId",
			errortypes.ErrInvalid,
			"Leave cannot be recorded against a {0} case",
			strings.ToLower(leaveCase.Status.Label()),
		)
	}

	entity := &worker.WorkerLeaveEntry{
		OrganizationID:           leaveCase.OrganizationID,
		BusinessUnitID:           leaveCase.BusinessUnitID,
		WorkerID:                 leaveCase.WorkerID,
		LeaveCaseID:              leaveCase.ID,
		UsedOn:                   req.UsedOn,
		Hours:                    req.Hours,
		CountsAgainstEntitlement: leaveCase.CountsAgainstEntitlement(),
		PTOID:                    req.PTOID,
		Notes:                    strings.TrimSpace(req.Notes),
		RecordedByID:             req.UserID,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return entity, nil
}

func (s *Service) PlanUpdateDay(ctx context.Context, req *UpdateDayRequest) (*EntryChange, error) {
	original, err := s.repo.GetEntryByID(ctx, &repositories.GetLeaveEntryByIDRequest{
		ID:         req.EntryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity := *original
	if req.Hours != nil {
		entity.Hours = *req.Hours
	}
	if req.Counts != nil {
		entity.CountsAgainstEntitlement = *req.Counts
	}
	if req.Notes != nil {
		entity.Notes = strings.TrimSpace(*req.Notes)
	}
	if !req.PTOID.IsNil() {
		entity.PTOID = req.PTOID
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &EntryChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanDeleteDay(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerLeaveEntry, error) {
	return s.repo.GetEntryByID(ctx, &repositories.GetLeaveEntryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}
