package accountingsynceffects

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type AuditEntry struct {
	Resource   permission.Resource
	ResourceID pulid.ID
	UserID     pulid.ID
	Tenant     pagination.TenantInfo
	Current    any
	Previous   map[string]any
	Comment    string
}

func (e *Effects) LogAudit(entry *AuditEntry) {
	params := &auditports.LogActionParams{
		Resource:       entry.Resource,
		ResourceID:     entry.ResourceID.String(),
		Operation:      permission.OpUpdate,
		UserID:         entry.UserID,
		CurrentState:   jsonutils.MustToJSON(entry.Current),
		PreviousState:  entry.Previous,
		OrganizationID: entry.Tenant.OrgID,
		BusinessUnitID: entry.Tenant.BuID,
	}
	if err := e.audit.LogAction(params, auditservice.WithComment(entry.Comment)); err != nil {
		e.l.Error("failed to log accounting sync audit", zap.Error(err))
	}
}
