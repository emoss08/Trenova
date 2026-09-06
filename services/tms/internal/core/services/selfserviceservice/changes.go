package selfserviceservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const eventProfileChangeReviewed = "dash.profile_change_reviewed"

func (s *Service) ListChangeRequests(
	ctx context.Context,
	req *repositories.ListProfileChangeRequestsRequest,
) ([]*worker.WorkerProfileChangeRequest, error) {
	return s.repo.ListChangeRequests(ctx, req)
}

func (s *Service) GetChangeRequest(
	ctx context.Context,
	req *repositories.GetProfileChangeRequestByIDRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	return s.repo.GetChangeRequestByID(ctx, req)
}

// SubmitChangeRequest is a driver asking for their record to change.
type SubmitChangeRequest struct {
	Worker     *worker.Worker
	Wanted     worker.ContactSnapshot
	Note       string
	TenantInfo pagination.TenantInfo
}

// SubmitChange records what the driver wants changed. A request that would
// change nothing is refused rather than queued — there is nothing for anybody
// to decide — and a second request while one is waiting is refused too,
// because a manager deciding two versions of the same address is a manager
// who will approve the wrong one.
func (s *Service) SubmitChange(
	ctx context.Context,
	req *SubmitChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	changes := worker.DiffContact(worker.ContactSnapshotOf(req.Worker), req.Wanted)
	if len(changes) == 0 {
		return nil, errortypes.NewValidationError(
			"changes",
			errortypes.ErrInvalid,
			"Nothing would change",
		)
	}

	pending, err := s.repo.ListChangeRequests(ctx, &repositories.ListProfileChangeRequestsRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.Worker.ID,
		Statuses:   []worker.ProfileChangeStatus{worker.ProfileChangePending},
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(pending) > 0 {
		return nil, errortypes.NewValidationError(
			"changes",
			errortypes.ErrInvalidOperation,
			"You already have a change waiting on your carrier — withdraw it to ask for something else",
		)
	}

	entity := &worker.WorkerProfileChangeRequest{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.Worker.ID,
		Status:         worker.ProfileChangePending,
		Changes:        changes,
		Note:           req.Note,
		SubmittedAt:    timeutils.NowUnix(),
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.CreateChangeRequest(ctx, entity)
}

// WithdrawChangeRequest is the driver taking their own request back.
type WithdrawChangeRequest struct {
	ID            pulid.ID
	ActorWorkerID pulid.ID
	TenantInfo    pagination.TenantInfo
}

func (s *Service) WithdrawChange(
	ctx context.Context,
	req *WithdrawChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	entity, err := s.repo.GetChangeRequestByID(
		ctx,
		&repositories.GetProfileChangeRequestByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if entity.WorkerID != req.ActorWorkerID {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"That is not your request",
		)
	}
	if !entity.Status.CanTransitionTo(worker.ProfileChangeWithdrawn) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("A %s request cannot be withdrawn", entity.Status),
		)
	}

	now := timeutils.NowUnix()
	entity.Status = worker.ProfileChangeWithdrawn
	entity.DecidedAt = &now

	return s.repo.UpdateChangeRequest(ctx, entity)
}

// DecideChangeRequest is the office answering.
type DecideChangeRequest struct {
	ID         pulid.ID
	Approve    bool
	Note       string
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Actor      *services.RequestActor
}

// DecideChange approves or rejects a request. An approval applies exactly the
// stored changes onto the worker — not whatever the driver's record looks
// like now, and not whatever else was on the form — then marks the request
// decided. Both are written before the driver is told, and a notification
// that fails does not undo either: the record changed, and the driver can see
// that it did.
func (s *Service) DecideChange(
	ctx context.Context,
	req *DecideChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	entity, err := s.repo.GetChangeRequestByID(
		ctx,
		&repositories.GetProfileChangeRequestByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	next := worker.ProfileChangeRejected
	if req.Approve {
		next = worker.ProfileChangeApproved
	}
	if !entity.Status.CanTransitionTo(next) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("A %s request cannot be %s", entity.Status, next),
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = next
	entity.DecidedAt = &now
	entity.DecidedByID = req.UserID
	entity.DecisionNote = req.Note

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if req.Approve {
		wrk, wErr := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
			ID:         entity.WorkerID,
			TenantInfo: req.TenantInfo,
		})
		if wErr != nil {
			return nil, wErr
		}
		entity.ApplyTo(wrk)
		if _, wErr = s.workerUpdater.Update(ctx, wrk, req.Actor); wErr != nil {
			return nil, wErr
		}
	}

	updated, err := s.repo.UpdateChangeRequest(ctx, entity)
	if err != nil {
		return nil, err
	}

	operation := permission.OpReject
	if req.Approve {
		operation = permission.OpApprove
	}
	s.audit(auditParams{
		resource:   permission.ResourceProfileChangeRequest,
		resourceID: updated.ID.String(),
		operation:  operation,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    req.Note,
	})

	s.notifyDriver(
		ctx,
		req.TenantInfo,
		updated.WorkerID,
		eventProfileChangeReviewed,
		documenttemplate.DriverNotificationContext{Approved: req.Approve, Reason: req.Note},
		map[string]any{"changeRequestId": updated.ID.String()},
	)

	return updated, nil
}
