package orgzonetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
)

type Organizations struct {
	tenantports.OrganizationRepository
	Timezone string
}

func (o Organizations) GetByID(
	_ context.Context,
	req tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	zone := o.Timezone
	if zone == "" {
		zone = "UTC"
	}
	return &tenant.Organization{ID: req.TenantInfo.OrgID, Timezone: zone}, nil
}
