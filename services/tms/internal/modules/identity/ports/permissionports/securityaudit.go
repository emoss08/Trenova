package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

type SecurityChange struct {
	Resource       permission.Resource
	ResourceID     string
	Operation      permission.Operation
	Actor          kernelports.AuditActor
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Before         any
	After          any
	Comment        string
	Metadata       map[string]any
}

type SecurityAuditor interface {
	RecordChange(ctx context.Context, change *SecurityChange)
}
