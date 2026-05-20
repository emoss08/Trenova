package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

type TenantSyncRepository interface {
	ListBusinessUnits(ctx context.Context) ([]services.TenantSyncBusinessUnit, error)
	ListOrganizations(ctx context.Context) ([]services.TenantSyncOrganization, error)
	ListBusinessUnitsByID(
		ctx context.Context,
		ids []pulid.ID,
	) ([]services.TenantSyncBusinessUnit, error)
	ListOrganizationsByID(
		ctx context.Context,
		ids []pulid.ID,
	) ([]services.TenantSyncOrganization, error)
}
