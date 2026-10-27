package ediservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (s *Service) logAction(
	entity edilading.Audited,
	actor *permissionports.RequestActor,
	operation permission.Operation,
	previous any,
	current any,
	comment string,
) {
	edilading.LogAction(s.auditService, s.l, &edilading.AuditEntry{
		Entity:    entity,
		Actor:     actor,
		Operation: operation,
		Previous:  previous,
		Current:   current,
		Comment:   comment,
	})
}
