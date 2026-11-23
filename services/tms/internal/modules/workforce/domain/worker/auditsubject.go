package worker

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (w *Worker) AuditSubject() (permission.Resource, pulid.ID) {
	if w == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, w.ID
}

func (wpto *WorkerPTO) AuditSubject() (permission.Resource, pulid.ID) {
	if wpto == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, wpto.WorkerID
}
