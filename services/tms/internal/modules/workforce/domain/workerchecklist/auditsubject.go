package workerchecklist

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (c *WorkerChecklist) AuditSubject() (permission.Resource, pulid.ID) {
	if c == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, c.WorkerID
}
