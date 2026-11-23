package workerschedule

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (a *WorkerShiftAssignment) AuditSubject() (permission.Resource, pulid.ID) {
	if a == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, a.WorkerID
}

func (p *WorkerAvailabilityPreference) AuditSubject() (permission.Resource, pulid.ID) {
	if p == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, p.WorkerID
}
