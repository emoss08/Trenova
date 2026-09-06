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
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerSafetyEvent, error) {
	return s.repo.ListEvents(ctx, &repositories.ListWorkerSafetyEventsRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

func (s *Service) GetEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	return s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeWorker:   true,
		IncludeDocument: true,
	})
}

// prepareEvent normalises a row, fills in derived fields and validates it.
func (s *Service) prepareEvent(ctx context.Context, entity *worker.WorkerSafetyEvent) error {
	entity.Description = strings.TrimSpace(entity.Description)
	entity.Location = strings.TrimSpace(entity.Location)
	entity.ReferenceNumber = strings.TrimSpace(entity.ReferenceNumber)
	entity.Resolution = strings.TrimSpace(entity.Resolution)
	if entity.Kind != worker.SafetyEventInspection {
		entity.InspectionResult = worker.InspectionResultNone
		entity.InspectionLevel = nil
		entity.OutOfService = false
	} else if entity.InspectionResult == worker.InspectionResultOutOfService {
		entity.OutOfService = true
	}
	if entity.Kind != worker.SafetyEventAccident {
		entity.Preventable = false
	}
	entity.DefaultPointsExpiry()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	if !entity.DocumentID.IsNil() {
		return s.requireWorkerDocument(ctx, eventTenant(entity), entity.WorkerID, entity.ID, entity.DocumentID)
	}
	return nil
}

func (s *Service) requireWorkerDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID, ownerID, documentID pulid.ID,
) error {
	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	ownedByRecord := !ownerID.IsNil() && doc.ResourceType == safetyResourceType && doc.ResourceID == ownerID.String()
	ownedByWorker := doc.ResourceType == workerResourceType && doc.ResourceID == workerID.String()
	if !ownedByRecord && !ownedByWorker {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}
	return nil
}

// CreateEvent records what happened. Points default from the kind and
// severity when the caller leaves them at zero and has not said otherwise.
func (s *Service) CreateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
	userID pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	log := s.l.With(zap.String("operation", "CreateEvent"), zap.String("workerId", entity.WorkerID.String()))

	if _, err := s.loadWorker(ctx, eventTenant(entity), entity.WorkerID); err != nil {
		return nil, err
	}
	entity.RecordedByID = userID
	if entity.Status == "" {
		entity.Status = worker.SafetyEventStatusOpen
	}
	if entity.Status == worker.SafetyEventStatusClosed {
		now := timeutils.NowUnix()
		entity.ClosedAt = &now
		entity.ClosedByID = userID
	}
	if err := s.prepareEvent(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateEvent(ctx, entity)
	if err != nil {
		log.Error("failed to create safety event", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: userID, tenant: eventTenant(created),
		current: created, comment: "Safety event recorded: " + created.Kind.String(), log: log,
	})
	s.publish(ctx, eventTenant(created), realtimeSafetyEvent, permission.OpCreate, created.ID, userID)
	s.refreshRollupQuietly(ctx, eventTenant(created), created.WorkerID)
	return created, nil
}

// UpdateEvent edits the narrative and the points; it never moves the status
// (Close does that so the resolution is always captured).
func (s *Service) UpdateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
	userID pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	log := s.l.With(zap.String("operation", "UpdateEvent"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:         entity.ID,
		TenantInfo: eventTenant(entity),
	})
	if err != nil {
		return nil, err
	}
	if entity.Version > 0 && original.Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Safety event was changed by someone else. Reload and try again",
		)
	}
	entity.WorkerID = original.WorkerID
	entity.Status = original.Status
	entity.RecordedByID = original.RecordedByID
	entity.ClosedByID = original.ClosedByID
	entity.ClosedAt = original.ClosedAt
	if entity.Resolution == "" {
		entity.Resolution = original.Resolution
	}
	entity.CreatedAt = original.CreatedAt
	entity.Version = original.Version
	if err = s.prepareEvent(ctx, entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateEvent(ctx, entity)
	if err != nil {
		log.Error("failed to update safety event", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: userID, tenant: eventTenant(updated),
		current: updated, previous: original, comment: "Safety event updated", log: log,
	})
	s.publish(ctx, eventTenant(updated), realtimeSafetyEvent, permission.OpUpdate, updated.ID, userID)
	s.refreshRollupQuietly(ctx, eventTenant(updated), updated.WorkerID)
	return updated, nil
}

type EventStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Resolution string
	Version    int64
	UserID     pulid.ID
}

// CloseEvent resolves an event; the resolution is required and kept.
func (s *Service) CloseEvent(
	ctx context.Context,
	req *EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	return s.moveEvent(ctx, req, worker.SafetyEventStatusClosed, permission.OpClose, "Safety event closed")
}

// ReviewEvent flags an event as being looked into.
func (s *Service) ReviewEvent(
	ctx context.Context,
	req *EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	return s.moveEvent(ctx, req, worker.SafetyEventStatusUnderReview, permission.OpUpdate, "Safety event under review")
}

// ReopenEvent puts a closed event back on the open list.
func (s *Service) ReopenEvent(
	ctx context.Context,
	req *EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	return s.moveEvent(ctx, req, worker.SafetyEventStatusOpen, permission.OpReopen, "Safety event reopened")
}

func (s *Service) moveEvent(
	ctx context.Context,
	req *EventStatusRequest,
	target worker.SafetyEventStatus,
	operation permission.Operation,
	comment string,
) (*worker.WorkerSafetyEvent, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
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
			"Safety event was changed by someone else. Reload and try again",
		)
	}
	if original.Status == target {
		return original, nil
	}

	updated := *original
	updated.Status = target
	switch target {
	case worker.SafetyEventStatusClosed:
		now := timeutils.NowUnix()
		updated.ClosedAt = &now
		updated.ClosedByID = req.UserID
		if resolution := strings.TrimSpace(req.Resolution); resolution != "" {
			updated.Resolution = resolution
		}
	case worker.SafetyEventStatusOpen, worker.SafetyEventStatusUnderReview:
		updated.ClosedAt = nil
		updated.ClosedByID = pulid.Nil
	}
	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.UpdateEvent(ctx, &updated)
	if err != nil {
		log.Error("failed to move safety event", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: saved.GetResourceID(),
		operation: operation, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: comment, log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeSafetyEvent, operation, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)
	return saved, nil
}

// DeleteEvent removes an event recorded in error. Closed events are history
// and stay.
func (s *Service) DeleteEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With(zap.String("operation", "DeleteEvent"), zap.String("id", id.String()))

	original, err := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if original.IsClosed() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Closed events are part of the record and cannot be deleted. Reopen it first",
		)
	}
	if err = s.repo.DeleteEvent(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete safety event", zap.Error(err))
		return err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerSafetyEvent, resourceID: original.GetResourceID(),
		operation: permission.OpDelete, userID: userID, tenant: tenantInfo,
		current: original, comment: "Safety event deleted", log: log,
	})
	s.publish(ctx, tenantInfo, realtimeSafetyEvent, permission.OpDelete, original.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, original.WorkerID)
	return nil
}
