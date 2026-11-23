package agentmemory

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
)

func (m *Memory) AuditSubject() (permission.Resource, pulid.ID) {
	if m == nil || m.SubjectID == nil || m.SubjectID.IsNil() {
		return "", pulid.Nil
	}
	resource, known := recordkind.ResourceOfIDPrefix(m.SubjectID.Prefix())
	if !known {
		return "", pulid.Nil
	}
	return resource, *m.SubjectID
}
