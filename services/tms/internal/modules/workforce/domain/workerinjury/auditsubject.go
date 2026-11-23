package workerinjury

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (i *WorkerInjury) AuditSubject() (permission.Resource, pulid.ID) {
	if i == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, i.WorkerID
}
