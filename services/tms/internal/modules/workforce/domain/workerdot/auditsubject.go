package workerdot

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (t *WorkerDOTTest) AuditSubject() (permission.Resource, pulid.ID) {
	if t == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, t.WorkerID
}

func (v *WorkerDOTViolation) AuditSubject() (permission.Resource, pulid.ID) {
	if v == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, v.WorkerID
}
