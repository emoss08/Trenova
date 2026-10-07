package orgzone

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type OrganizationReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

func Location(
	ctx context.Context,
	organizations OrganizationReader,
	info pagination.TenantInfo,
) (*time.Location, error) {
	org, err := organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: info,
	})
	if err != nil {
		return nil, fmt.Errorf("read the organization's timezone: %w", err)
	}

	return timeutils.LoadLocation(org.Timezone), nil
}
