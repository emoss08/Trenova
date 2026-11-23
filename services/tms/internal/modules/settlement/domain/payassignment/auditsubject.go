package payassignment

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (a *WorkerPayAssignment) AuditSubject() (permission.Resource, pulid.ID) {
	if a == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, a.WorkerID
}
