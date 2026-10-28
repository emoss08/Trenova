package timesheetresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/shared/pulid"
)

// timesheetPermissionFor maps the answer onto the grant it needs. Handing a
// week over is not the same act as signing it off, and a carrier that lets
// everybody submit still wants approval held to a shorter list.
func timesheetPermissionFor(status timesheet.TimesheetStatus) permission.Operation {
	switch status {
	case timesheet.TimesheetSubmitted:
		return permission.OpSubmit
	case timesheet.TimesheetApproved:
		return permission.OpApprove
	case timesheet.TimesheetRejected:
		return permission.OpReject
	case timesheet.TimesheetOpen:
		// Taking an approval back is the approver's act.
		return permission.OpApprove
	case timesheet.TimesheetLocked:
		return permission.OpExport
	default:
		return permission.OpApprove
	}
}

// mustOptionalID is optionalID for a value the caller has already accepted may
// be absent and cannot be malformed in a way worth a separate message.
func mustOptionalID(value *string) pulid.ID {
	id, err := base.OptionalID(value)
	if err != nil {
		return pulid.Nil
	}
	return id
}
