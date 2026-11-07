package auditexportservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/dbscope"
)

func withTenant(ctx context.Context, export *audit.Export) context.Context {
	return dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: export.OrganizationID,
		BusinessUnitID: export.BusinessUnitID,
	})
}
