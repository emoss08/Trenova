package workersafetyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

	entity, err := s.PlanRecordViolation(ctx, req)
	if err != nil {
		return nil, err
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

	change, err := s.PlanUpdateViolation(ctx, req)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateViolation(ctx, change.After)
	if err != nil {
		log.Error("failed to update safety violation", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: req.UserID, tenant: req.TenantInfo,
		current: updated, previous: change.Before, comment: "Violation corrected", log: log,
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

	original, err := s.PlanDeleteViolation(ctx, tenantInfo, id)
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
