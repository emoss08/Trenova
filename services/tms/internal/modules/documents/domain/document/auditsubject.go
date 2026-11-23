package document

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Document) AuditSubject() (permission.Resource, pulid.ID) {
	if d == nil || d.ResourceType == "" || d.ResourceID == "" {
		return "", pulid.Nil
	}
	return permission.Resource(d.ResourceType), pulid.ID(d.ResourceID)
}
