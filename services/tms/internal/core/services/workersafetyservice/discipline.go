package workersafetyservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workeremploymentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListActions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerDisciplinaryAction, error) {
	return s.repo.ListActions(ctx, &repositories.ListWorkerDisciplinaryActionsRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		IncludeEvent:  true,
		IncludeActors: true,
	})
}

// Ladder is where the worker stands and what the next rung would be.
func (s *Service) Ladder(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.DisciplinaryLadder, error) {
	actions, err := s.repo.ListActions(ctx, &repositories.ListWorkerDisciplinaryActionsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	return worker.BuildDisciplinaryLadder(actions, timeutils.NowUnix()), nil
}

type IssueActionRequest struct {
	TenantInfo     pagination.TenantInfo
	WorkerID       pulid.ID
	Level          worker.DisciplinaryLevel
	Reason         string
	Details        string
	OccurredAt     *int64
	ExpiresAt      *int64
	SuspensionDays *int32
	SafetyEventID  pulid.ID
	DocumentID     pulid.ID
	// RecordEmploymentEvent puts a Suspended / Terminated event on the
	// timeline so dispatch and pay follow; off for a suspension already served.
	RecordEmploymentEvent bool
	UserID                pulid.ID
}

// IssueActionResult is the action plus what it did to employment.
type IssueActionResult struct {
	Action          *worker.WorkerDisciplinaryAction
	EmploymentEvent *worker.WorkerEmploymentEvent
}

// IssueAction records a rung on the ladder. Suspensions and terminations can
// also record the matching employment event so status moves through the
// timeline like everything else; the driver is told either way.
func (s *Service) IssueAction(
	ctx context.Context,
	req *IssueActionRequest,
) (*IssueActionResult, error) {
	log := s.l.With(
		zap.String("operation", "IssueAction"),
		zap.String("workerId", req.WorkerID.String()),
		zap.String("level", req.Level.String()),
	)

	wrk, err := s.loadWorker(ctx, req.TenantInfo, req.WorkerID)
	if err != nil {
		return nil, err
	}
	if !req.SafetyEventID.IsNil() {
		event, eventErr := s.repo.GetEventByID(ctx, &repositories.GetWorkerSafetyEventByIDRequest{
			ID:         req.SafetyEventID,
			TenantInfo: req.TenantInfo,
		})
		if eventErr != nil {
			return nil, eventErr
		}
		if event.WorkerID != req.WorkerID {
			return nil, errortypes.NewValidationError(
				"safetyEventId",
				errortypes.ErrInvalid,
				"That event belongs to another worker",
			)
		}
	}

	now := timeutils.NowUnix()
	entity := &worker.WorkerDisciplinaryAction{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		Level:          req.Level,
		Status:         worker.DisciplinaryStatusActive,
		Reason:         strings.TrimSpace(req.Reason),
		Details:        strings.TrimSpace(req.Details),
		OccurredAt:     req.OccurredAt,
		IssuedAt:       now,
		ExpiresAt:      req.ExpiresAt,
		SuspensionDays: req.SuspensionDays,
		SafetyEventID:  req.SafetyEventID,
		DocumentID:     req.DocumentID,
		IssuedByID:     req.UserID,
	}
	entity.DefaultExpiry()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if !entity.DocumentID.IsNil() {
		if err = s.requireWorkerDocument(ctx, req.TenantInfo, req.WorkerID, pulid.Nil, entity.DocumentID); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.CreateAction(ctx, entity)
	if err != nil {
		log.Error("failed to issue disciplinary action", zap.Error(err))
		return nil, err
	}
	result := &IssueActionResult{Action: created}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerDisciplinaryAction, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: req.UserID, tenant: req.TenantInfo,
		current: created, comment: "Disciplinary action issued: " + created.Level.String(), log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeDiscipline, permission.OpCreate, created.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, created.WorkerID)

	if req.RecordEmploymentEvent && s.employment != nil {
		kind := worker.EmploymentEventKind("")
		switch req.Level {
		case worker.DisciplinaryLevelSuspension:
			kind = worker.EmploymentEventSuspended
		case worker.DisciplinaryLevelTermination:
			kind = worker.EmploymentEventTerminated
		case worker.DisciplinaryLevelCoaching, worker.DisciplinaryLevelVerbalWarning,
			worker.DisciplinaryLevelWrittenWarning, worker.DisciplinaryLevelFinalWarning:
		}
		if kind != "" {
			recorded, recordErr := s.employment.Record(ctx, &workeremploymentservice.RecordRequest{
				TenantInfo:  req.TenantInfo,
				WorkerID:    req.WorkerID,
				Kind:        kind,
				EffectiveAt: now,
				Reason:      created.Reason,
				Notes:       disciplineReasonNote,
				DocumentID:  created.DocumentID,
				UserID:      req.UserID,
			})
			if recordErr != nil {
				return nil, fmt.Errorf("record employment event for %s: %w", req.Level, recordErr)
			}
			result.EmploymentEvent = recorded.Event
		}
	}

	if !wrk.UserID.IsNil() {
		s.notifyDriver(ctx, req.TenantInfo, req.WorkerID, eventDisciplinary, notification.PriorityHigh,
			documenttemplate.DriverNotificationContext{
				DisciplineLevel:  created.Level.String(),
				DisciplineReason: created.Reason,
			},
			fmt.Sprintf("discipline-%s", created.ID),
		)
	}
	return result, nil
}

type ActionStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Reason     string
	Version    int64
	UserID     pulid.ID
}

// RescindAction takes an action off the ladder with a reason; the row stays
// so the audit trail does.
func (s *Service) RescindAction(
	ctx context.Context,
	req *ActionStatusRequest,
) (*worker.WorkerDisciplinaryAction, error) {
	log := s.l.With(zap.String("operation", "RescindAction"), zap.String("id", req.ID.String()))

	original, err := s.loadAction(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if original.Status == worker.DisciplinaryStatusRescinded {
		return original, nil
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the action is rescinded",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.DisciplinaryStatusRescinded
	updated.RescindedAt = &now
	updated.RescindedByID = req.UserID
	updated.RescindReason = reason
	saved, err := s.repo.UpdateAction(ctx, &updated)
	if err != nil {
		log.Error("failed to rescind disciplinary action", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerDisciplinaryAction, resourceID: saved.GetResourceID(),
		operation: permission.OpCancel, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: "Disciplinary action rescinded: " + reason, log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeDiscipline, permission.OpCancel, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)
	return saved, nil
}

// AcknowledgeAction is the driver confirming they have read the action.
func (s *Service) AcknowledgeAction(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	workerID pulid.ID,
	comment string,
) (*worker.WorkerDisciplinaryAction, error) {
	original, err := s.loadAction(ctx, tenantInfo, id, 0)
	if err != nil {
		return nil, err
	}
	if original.WorkerID != workerID {
		return nil, errortypes.NewNotFoundError("Disciplinary action not found")
	}
	if original.IsAcknowledged() {
		return original, nil
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.AcknowledgedAt = &now
	updated.WorkerComment = strings.TrimSpace(comment)
	saved, err := s.repo.UpdateAction(ctx, &updated)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, tenantInfo, realtimeDiscipline, permission.OpUpdate, saved.ID, pulid.Nil)
	s.refreshRollupQuietly(ctx, tenantInfo, saved.WorkerID)
	return saved, nil
}

func (s *Service) loadAction(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	version int64,
) (*worker.WorkerDisciplinaryAction, error) {
	original, err := s.repo.GetActionByID(ctx, &repositories.GetWorkerDisciplinaryActionByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if version > 0 && original.Version != version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Disciplinary action was changed by someone else. Reload and try again",
		)
	}
	return original, nil
}
