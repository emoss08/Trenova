package paybenefit

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (e *WorkerBenefitEnrollment) AuditSubject() (permission.Resource, pulid.ID) {
	if e == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, e.WorkerID
}
