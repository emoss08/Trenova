package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

// requireTimesheetWorkerScope is the gate every action aimed at one worker's
// hours goes through: the grant, then the reporting line. A timesheet is a
// wage record, so a manager scoped to their own team must not reach somebody
// else's.
func (r *Resolver) RequireTimesheetWorkerScope(
	ctx context.Context,
	operation permission.Operation,
	workerID string,
) (*authctx.AuthContext, pulid.ID, error) {
	id, err := pulid.MustParse(workerID)
	if err != nil {
		return nil, pulid.Nil, InvalidIDError("workerId", "Worker is invalid")
	}

	authCtx, _, err := r.RequireTeamScope(
		ctx,
		permission.ResourceTimesheet,
		operation,
		worker.ApprovalScopeAll,
		id,
	)
	if err != nil {
		return nil, pulid.Nil, err
	}

	return authCtx, id, nil
}
