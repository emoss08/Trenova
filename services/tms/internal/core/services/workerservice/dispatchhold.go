package workerservice

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

// DispatchHoldRequest takes a driver off the board, or puts them back.
type DispatchHoldRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	// Held is what canBeAssigned becomes: true takes them off dispatch.
	Held   bool
	Reason string
	Actor  *services.RequestActor
}

/*
SetDispatchHold stops a driver being given new freight, or lets them be given
it again.

It is deliberately narrower than Update: employment, pay and profile are
untouched, and nothing happens to the load the driver is on. A credential
that has lapsed means the next load cannot be theirs — not that they are off
the payroll, and not that the truck they are in turns around. Lifting the
hold is the same call with Held false, which is what makes it something a
person can undo without reconstructing the record.
*/
func (s *Service) SetDispatchHold(
	ctx context.Context,
	req DispatchHoldRequest,
) (*worker.Worker, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, errDispatchHoldReason()
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         req.WorkerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = ApplyDispatchHold(entity, &req); err != nil {
		return nil, err
	}

	return s.UpdateFromEmployment(ctx, entity, req.Actor)
}

func ApplyDispatchHold(entity *worker.Worker, req *DispatchHoldRequest) error {
	if strings.TrimSpace(req.Reason) == "" {
		return errDispatchHoldReason()
	}
	if entity.CanBeAssigned != req.Held {
		return errortypes.NewValidationError("held", errortypes.ErrInvalidOperation,
			holdNoOpMessage(req.Held))
	}

	entity.CanBeAssigned = !req.Held

	return nil
}

func errDispatchHoldReason() error {
	return errortypes.NewValidationError("reason", errortypes.ErrRequired,
		"Say why this driver is being held")
}

func holdNoOpMessage(held bool) string {
	if held {
		return "This driver is already off dispatch"
	}

	return "This driver is already on dispatch"
}
