package shipmentboardbrief

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orgzone"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Day struct {
	Date     string
	Timezone string
}

func Today(
	ctx context.Context,
	organizations orgzone.OrganizationReader,
	tenantInfo pagination.TenantInfo,
) (Day, error) {
	location, err := orgzone.Location(ctx, organizations, tenantInfo)
	if err != nil {
		return Day{}, err
	}

	timezone := location.String()

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
