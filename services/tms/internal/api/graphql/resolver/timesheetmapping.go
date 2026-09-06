package resolver

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
func (r *Resolver) requireTimesheetWorkerScope(
	ctx context.Context,
	operation permission.Operation,
	workerID string,
) (*authctx.AuthContext, pulid.ID, error) {
	id, err := pulid.MustParse(workerID)
	if err != nil {
		return nil, pulid.Nil, invalidIDError("workerId", "Worker is invalid")
	}

	authCtx, _, err := r.requireTeamScope(
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

// timesheetPermissionFor maps the answer onto the grant it needs. Handing a
// week over is not the same act as signing it off, and a carrier that lets
// everybody submit still wants approval held to a shorter list.
func timesheetPermissionFor(status worker.TimesheetStatus) permission.Operation {
	switch status {
	case worker.TimesheetSubmitted:
		return permission.OpSubmit
	case worker.TimesheetApproved:
		return permission.OpApprove
	case worker.TimesheetRejected:
		return permission.OpReject
	case worker.TimesheetOpen:
		// Taking an approval back is the approver's act.
		return permission.OpApprove
	case worker.TimesheetLocked:
		return permission.OpExport
	default:
		return permission.OpApprove
	}
}

// mustOptionalID is optionalID for a value the caller has already accepted may
// be absent and cannot be malformed in a way worth a separate message.
func mustOptionalID(value *string) pulid.ID {
	id, err := optionalID(value)
	if err != nil {
		return pulid.Nil
	}
	return id
}
