package workerleave

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (c *WorkerLeaveCase) AuditSubject() (permission.Resource, pulid.ID) {
	if c == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, c.WorkerID
}

func (e *WorkerLeaveEntry) AuditSubject() (permission.Resource, pulid.ID) {
	if e == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, e.WorkerID
}
