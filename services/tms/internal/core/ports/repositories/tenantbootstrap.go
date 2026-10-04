package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type BootstrapTenantRequest struct {
	BusinessUnitName  string
	Organization      *tenant.Organization
	LoginSlugBase     string
	StateAbbreviation string
	Owner             *tenant.User
	UsernameBase      string
	Now               int64
}

type BootstrapTenantResult struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	Owner        *tenant.User
	AdminRoleID  pulid.ID
}

type TenantBootstrapRepository interface {
	LockProvisioning(ctx context.Context) error
	Bootstrap(ctx context.Context, req *BootstrapTenantRequest) (*BootstrapTenantResult, error)
}
