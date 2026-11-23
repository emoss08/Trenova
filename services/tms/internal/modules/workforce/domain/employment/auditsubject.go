package employment

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (e *WorkerEmploymentEvent) AuditSubject() (permission.Resource, pulid.ID) {
	if e == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, e.WorkerID
}

func (v *WorkerEmploymentVerification) AuditSubject() (permission.Resource, pulid.ID) {
	if v == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, v.WorkerID
}
