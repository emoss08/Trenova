package timesheet

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (t *Timesheet) AuditSubject() (permission.Resource, pulid.ID) {
	if t == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, t.WorkerID
}

func (e *TimeClockEntry) AuditSubject() (permission.Resource, pulid.ID) {
	if e == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, e.WorkerID
}
