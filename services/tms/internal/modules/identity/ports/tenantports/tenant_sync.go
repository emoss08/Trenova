package tenantports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type TenantSyncRepository interface {
	ListBusinessUnits(ctx context.Context) ([]tenant.SyncBusinessUnit, error)
	ListOrganizations(ctx context.Context) ([]tenant.SyncOrganization, error)
	ListBusinessUnitsByID(
		ctx context.Context,
		ids []pulid.ID,
	) ([]tenant.SyncBusinessUnit, error)
	ListOrganizationsByID(
		ctx context.Context,
		ids []pulid.ID,
	) ([]tenant.SyncOrganization, error)
	// ListTenantsPage lists, across every tenant, the organizations after the
	// cursor in id order, at most limit: a fan-out pages through them rather
	// than reading every organization at once.
	ListTenantsPage(ctx context.Context, after pulid.ID, limit int) ([]pagination.TenantInfo, error)
}
