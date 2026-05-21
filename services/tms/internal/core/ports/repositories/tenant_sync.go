package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
}

type TenantProvisioningRepository interface {
	UpsertProvisioningSnapshot(
		ctx context.Context,
		req *tenant.ProvisioningRequest,
	) (*tenant.ProvisioningResult, error)
}
