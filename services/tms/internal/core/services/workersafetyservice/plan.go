package workersafetyservice

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

type (
	EventChange     = services.RecordChange[worker.WorkerSafetyEvent]
	ViolationChange = services.RecordChange[worker.WorkerSafetyViolation]
)

// PlanCreateEvent is the event CreateEvent would record, checked and filled
// in the same way, without writing it.
func (s *Service) PlanCreateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
	userID pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	planned := *entity
	if err := s.prepareCreateEvent(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

// PlanUpdateEvent is what UpdateEvent would leave the event as.
func (s *Service) PlanUpdateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
) (*EventChange, error) {
	planned := *entity
	return s.planUpdateEvent(ctx, &planned)
}

func (s *Service) PlanCloseEvent(ctx context.Context, req *EventStatusRequest) (*EventChange, error) {
	return s.planMoveEvent(ctx, req, worker.SafetyEventStatusClosed)
}

func (s *Service) PlanReviewEvent(
	ctx context.Context,
	req *EventStatusRequest,
) (*EventChange, error) {
	return s.planMoveEvent(ctx, req, worker.SafetyEventStatusUnderReview)
}

func (s *Service) PlanReopenEvent(
	ctx context.Context,
	req *EventStatusRequest,
) (*EventChange, error) {
	return s.planMoveEvent(ctx, req, worker.SafetyEventStatusOpen)
}

// PlanDeleteEvent is the event DeleteEvent would remove, refused as the
// delete refuses it.
func (s *Service) PlanDeleteEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	original, err := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.IsClosed() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Closed events are part of the record and cannot be deleted. Reopen it first",
		)
	}
	return original, nil
}

// PlanGiveRecognition is the recognition GiveRecognition would record.
func (s *Service) PlanGiveRecognition(
	ctx context.Context,
	entity *worker.WorkerRecognition,
	userID pulid.ID,
) (*worker.WorkerRecognition, error) {
	planned := *entity
	if _, err := s.prepareRecognition(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) PlanDeleteRecognition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerRecognition, error) {
	return s.repo.GetRecognitionByID(ctx, &repositories.GetWorkerRecognitionByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

// PlanRecordViolation is the violation RecordViolation would cite, with the
// worker taken from the event and the BASIC filled in.
func (s *Service) PlanRecordViolation(
	ctx context.Context,
	req *RecordViolationRequest,
) (*worker.WorkerSafetyViolation, error) {
	event, err := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:         req.SafetyEventID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity := &worker.WorkerSafetyViolation{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		SafetyEventID:  event.ID,
		WorkerID:       event.WorkerID,
		Basic:          req.Basic,
		Code:           strings.TrimSpace(req.Code),
		Description:    strings.TrimSpace(req.Description),
		SeverityWeight: req.SeverityWeight,
		OutOfService:   req.OutOfService,
	}
	if entity.SeverityWeight <= 0 {
		entity.SeverityWeight = 1
	}
	if entity.Basic == "" {
		entity.Basic = worker.SuggestedBasic(event.Kind, event.InspectionResult)
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return entity, nil
}

func (s *Service) PlanUpdateViolation(
	ctx context.Context,
	req *UpdateViolationRequest,
) (*ViolationChange, error) {
	original, err := s.repo.GetViolationByID(ctx, &repositories.GetWorkerSafetyViolationByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Violation was changed by someone else. Reload and try again",
		)
	}

	entity := *original
	if req.Basic != "" {
		entity.Basic = req.Basic
	}
	entity.Code = strings.TrimSpace(req.Code)
	entity.Description = strings.TrimSpace(req.Description)
	if req.SeverityWeight > 0 {
		entity.SeverityWeight = req.SeverityWeight
	}
	entity.OutOfService = req.OutOfService

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &ViolationChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanDeleteViolation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerSafetyViolation, error) {
	return s.repo.GetViolationByID(ctx, &repositories.GetWorkerSafetyViolationByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}
