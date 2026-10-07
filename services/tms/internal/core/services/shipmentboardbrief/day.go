package shipmentboardbrief

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Day struct {
	Date     string
	Timezone string
}

func Today(
	ctx context.Context,
	organizations repositories.OrganizationCacheRepository,
	tenantInfo pagination.TenantInfo,
) (Day, error) {
	organization, err := organizations.GetByID(ctx, tenantInfo.OrgID)
	if err != nil {
		return Day{}, fmt.Errorf("load organization: %w", err)
	}

	timezone := timeutils.NormalizeTimezone(organization.Timezone)

	return Day{Date: timeutils.CurrentDateInTimezone(timezone), Timezone: timezone}, nil
}

func Latest(
	ctx context.Context,
	briefs repositories.ShipmentBriefRepository,
	tenantInfo pagination.TenantInfo,
	day Day,
) (*shipmentbrief.Brief, error) {
	return briefs.Latest(ctx, &repositories.GetLatestShipmentBriefRequest{
		TenantInfo: tenantInfo,
		BriefDate:  day.Date,
	})
}
