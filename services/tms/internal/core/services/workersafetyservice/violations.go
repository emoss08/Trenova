package workersafetyservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const realtimeViolation = "worker_safety_violation"

func (s *Service) ListViolations(
	ctx context.Context,
	req *repositories.ListWorkerSafetyViolationsRequest,
) ([]*worker.WorkerSafetyViolation, error) {
	return s.repo.ListViolations(ctx, req)
}

// RecordViolationRequest cites one violation on an existing safety event.
type RecordViolationRequest struct {
	TenantInfo     pagination.TenantInfo
	SafetyEventID  pulid.ID
	Basic          worker.CSABasic
	Code           string
	Description    string
	SeverityWeight int16
	OutOfService   bool
	UserID         pulid.ID
}

// RecordViolation cites a violation against an event. The worker is copied
// from the event rather than accepted from the caller: a violation attached to
// one driver's inspection and counted against another's record would be
// invisible and wrong.
func (s *Service) RecordViolation(
	ctx context.Context,
	req *RecordViolationRequest,
) (*worker.WorkerSafetyViolation, error) {
	log := s.l.With(
		zap.String("operation", "RecordViolation"),
		zap.String("eventId", req.SafetyEventID.String()),
	)

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
	// An unstated BASIC falls back to what the event itself implies, so a clerk
	// keying an inspection does not have to classify every line to record one.
	if entity.Basic == "" {
		entity.Basic = worker.SuggestedBasic(event.Kind, event.InspectionResult)
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateViolation(ctx, entity)
	if err != nil {
		log.Error("failed to record safety violation", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: req.UserID, tenant: req.TenantInfo,
		current: created,
		comment: "Violation cited against " + created.Basic.String(),
		log:     log,
	})
	s.publish(ctx, req.TenantInfo, realtimeViolation, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

// UpdateViolationRequest corrects a cited violation.
type UpdateViolationRequest struct {
	TenantInfo     pagination.TenantInfo
	ID             pulid.ID
	Basic          worker.CSABasic
	Code           string
	Description    string
	SeverityWeight int16
	OutOfService   bool
	Version        int64
	UserID         pulid.ID
}

func (s *Service) UpdateViolation(
	ctx context.Context,
	req *UpdateViolationRequest,
) (*worker.WorkerSafetyViolation, error) {
	log := s.l.With(zap.String("operation", "UpdateViolation"), zap.String("id", req.ID.String()))

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

	previous := *original
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

	updated, err := s.repo.UpdateViolation(ctx, &entity)
	if err != nil {
		log.Error("failed to update safety violation", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: req.UserID, tenant: req.TenantInfo,
		current: updated, previous: &previous, comment: "Violation corrected", log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeViolation, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

func (s *Service) DeleteViolation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With(zap.String("operation", "DeleteViolation"), zap.String("id", id.String()))

	original, err := s.repo.GetViolationByID(ctx, &repositories.GetWorkerSafetyViolationByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.DeleteViolation(ctx, &repositories.GetWorkerSafetyViolationByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete safety violation", zap.Error(err))
		return err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: original.GetResourceID(),
		operation: permission.OpDelete, userID: userID, tenant: tenantInfo,
		current: original, comment: "Violation removed", log: log,
	})
	s.publish(ctx, tenantInfo, realtimeViolation, permission.OpDelete, id, userID)

	return nil
}
