package workertraining

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *WorkerTrainingRecord) AuditSubject() (permission.Resource, pulid.ID) {
	if r == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, r.WorkerID
}
