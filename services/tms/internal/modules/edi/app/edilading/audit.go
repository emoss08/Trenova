package edilading

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type Audited interface {
	GetID() pulid.ID
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

type AuditEntry struct {
	Entity    Audited
	Actor     *permissionports.RequestActor
	Operation permission.Operation
	Previous  any
	Current   any
	Comment   string
}

func LogAction(service auditports.AuditService, logger *zap.Logger, entry *AuditEntry) {
	if service == nil || entry.Actor == nil || entry.Entity == nil {
		return
	}
	auditActor := entry.Actor.AuditActor()
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceEDI,
		ResourceID:     entry.Entity.GetID().String(),
		Operation:      entry.Operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: entry.Actor.OrganizationID,
		BusinessUnitID: entry.Actor.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(entry.Current),
	}
	if entry.Previous != nil {
		params.PreviousState = jsonutils.MustToJSON(entry.Previous)
	}
	if err := service.LogAction(params, auditservice.WithComment(entry.Comment)); err != nil {
		logger.Warn("failed to log EDI audit action", zap.Error(err))
	}
}
