package tenantports

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type OrganizationReader interface {
	GetByID(ctx context.Context, req GetOrganizationByIDRequest) (*tenant.Organization, error)
}

func OrganizationLocation(
	ctx context.Context,
	organizations OrganizationReader,
	info pagination.TenantInfo,
) (*time.Location, error) {
	org, err := organizations.GetByID(ctx, GetOrganizationByIDRequest{TenantInfo: info})
	if err != nil {
		return nil, fmt.Errorf("read the organization's timezone: %w", err)
	}
	if org == nil {
		return time.UTC, nil
	}
	return timeutils.LoadLocation(org.Timezone), nil
}
