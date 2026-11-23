package pto

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func (a *WorkerPTOPolicyAssignment) AuditSubject() (permission.Resource, pulid.ID) {
	if a == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, a.WorkerID
}

func (e *WorkerPTOLedgerEntry) AuditSubject() (permission.Resource, pulid.ID) {
	if e == nil {
		return "", pulid.Nil
	}
	return permission.ResourceWorker, e.WorkerID
}
