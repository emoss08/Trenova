package orgstructureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Two rosters hold positions. A driving title is held by a worker, who must
// carry a CDL and everything a driver's record carries; a front-office title
// is held by somebody who logs in, through their membership in the
// organisation. Putting a dispatcher on a driving title, or a driver on a
// dispatch desk, would put a person on the wrong roster, so both refuse.

func (s *Service) ListPositionHolders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	positionID pulid.ID,
) ([]repositories.PositionHolderRow, error) {
	return s.repo.ListPositionHolders(ctx, tenantInfo, positionID)
}

// AssignRequest puts one person on a position, or takes them off one with a
// nil position.
type AssignRequest struct {
	TenantInfo pagination.TenantInfo
	// HolderID is the worker or the user, depending on the call.
	HolderID   pulid.ID
	PositionID pulid.ID
	UserID     pulid.ID
}

func (s *Service) AssignWorkerPosition(ctx context.Context, req *AssignRequest) error {
	position, err := s.assignablePosition(ctx, req, true)
	if err != nil {
		return err
	}
	if err = s.repo.SetWorkerPosition(ctx, &repositories.SetWorkerPositionRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.HolderID,
		PositionID: req.PositionID,
	}); err != nil {
		return err
	}
	s.auditAssignment(ctx, req, position, "Worker")
	return nil
}

func (s *Service) AssignUserPosition(ctx context.Context, req *AssignRequest) error {
	position, err := s.assignablePosition(ctx, req, false)
	if err != nil {
		return err
	}
	if err = s.repo.SetUserPosition(ctx, &repositories.SetUserPositionRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.HolderID,
		PositionID: req.PositionID,
	}); err != nil {
		return err
	}
	s.auditAssignment(ctx, req, position, "User")
	return nil
}

// assignablePosition is the position a holder may be put on: it exists, it is
// open, and it is on the right roster. Taking somebody off a position needs
// none of that, so a nil id comes back as nil.
func (s *Service) assignablePosition(
	ctx context.Context,
	req *AssignRequest,
	driving bool,
) (*worker.JobPosition, error) {
	if req.HolderID.IsNil() {
		return nil, errortypes.NewValidationError(
			"holderId",
			errortypes.ErrRequired,
			"Say who is being put on the position",
		)
	}
	if req.PositionID.IsNil() {
		return nil, nil
	}
	position, err := s.repo.GetPositionByID(ctx, &repositories.GetJobPositionByIDRequest{
		ID:         req.PositionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if position.Status != "Active" {
		return nil, errortypes.NewValidationError(
			"positionId",
			errortypes.ErrInvalidOperation,
			"That position is archived and cannot be filled",
		)
	}
	if position.IsDrivingPosition != driving {
		if driving {
			return nil, errortypes.NewValidationError(
				"positionId",
				errortypes.ErrInvalidOperation,
				"A worker holds a driving position; that title is held by users",
			)
		}
		return nil, errortypes.NewValidationError(
			"positionId",
			errortypes.ErrInvalidOperation,
			"A user holds a front-office position; that title needs a driver",
		)
	}
	return position, nil
}

func (s *Service) auditAssignment(
	ctx context.Context,
	req *AssignRequest,
	position *worker.JobPosition,
	holder string,
) {
	if position == nil {
		s.publish(
			ctx,
			req.TenantInfo,
			realtimePosition,
			permission.OpUpdate,
			req.PositionID,
			req.UserID,
		)
		return
	}
	s.audit(&auditParams{
		resource: permission.ResourceJobPosition, resourceID: position.GetResourceID(),
		operation: permission.OpUpdate, userID: req.UserID, tenant: req.TenantInfo,
		current: position,
		comment: holder + " " + req.HolderID.String() + " put on " + position.Title,
	})
	s.publish(
		ctx,
		req.TenantInfo,
		realtimePosition,
		permission.OpUpdate,
		position.ID,
		req.UserID,
	)
}
