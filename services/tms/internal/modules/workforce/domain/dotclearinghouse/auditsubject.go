package dotclearinghouse

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (q *WorkerClearinghouseQuery) AuditSubject() (permission.Resource, pulid.ID) {
	if q == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, q.WorkerID
}
