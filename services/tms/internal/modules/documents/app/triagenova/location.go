package triagenova

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type OrganizationReader interface {
	GetByID(
		ctx context.Context,
		req tenantports.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

func Organization(
	ctx context.Context,
	organizations OrganizationReader,
	tenantInfo pagination.TenantInfo,
) (*tenant.Organization, *time.Location, error) {
	org, err := organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, time.UTC, err
	}
	if !timeutils.IsValidLocation(org.Timezone) {
		return org, time.UTC, nil
	}

	location, err := time.LoadLocation(org.Timezone)
	if err != nil {
		return org, time.UTC, nil
	}

	return org, location, nil
}
