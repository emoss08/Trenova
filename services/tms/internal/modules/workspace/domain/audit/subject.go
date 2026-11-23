package audit

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MetadataSubjectResource = "subjectResource"
	MetadataSubjectID       = "subjectId"
)

const maxSubjectIDLength = 100

var fileSubjects = map[permission.Resource]struct{}{
	permission.ResourceWorker: {},
}

func KeepsFile(resource permission.Resource) bool {
	_, ok := fileSubjects[resource]
	return ok
}

type SubjectOwner interface {
	AuditSubject() (permission.Resource, pulid.ID)
}

func (e *Entry) SetSubject(resource permission.Resource, id pulid.ID) error {
	if !KeepsFile(resource) {
		return fmt.Errorf("audit entries do not record %q files", resource)
	}
	if id.IsNil() {
		return fmt.Errorf("an audit entry's %q subject needs an id", resource)
	}
	if len(id.String()) > maxSubjectIDLength {
		return fmt.Errorf(
			"audit subject id %q is longer than %d characters",
			id,
			maxSubjectIDLength,
		)
	}
	if e.Metadata == nil {
		e.Metadata = make(map[string]any, 2)
	}
	e.Metadata[MetadataSubjectResource] = resource.String()
	e.Metadata[MetadataSubjectID] = id.String()
	e.projectSubject()
	return nil
}

func (e *Entry) projectSubject() {
	resource, _ := e.Metadata[MetadataSubjectResource].(string)
	id, _ := e.Metadata[MetadataSubjectID].(string)
	if id == "" || len(id) > maxSubjectIDLength || !KeepsFile(permission.Resource(resource)) {
		e.SubjectResource = ""
		e.SubjectID = pulid.Nil
		return
	}
	e.SubjectResource = permission.Resource(resource)
	e.SubjectID = pulid.ID(id)
}
