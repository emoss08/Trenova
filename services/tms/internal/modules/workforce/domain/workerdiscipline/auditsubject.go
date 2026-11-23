package workerdiscipline

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (a *WorkerDisciplinaryAction) AuditSubject() (permission.Resource, pulid.ID) {
	if a == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, a.WorkerID
}

func (r *WorkerRecognition) AuditSubject() (permission.Resource, pulid.ID) {
	if r == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, r.WorkerID
}
