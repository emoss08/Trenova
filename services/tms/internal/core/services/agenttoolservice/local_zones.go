package agenttoolservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type locationZoneReader interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetLocationsByIDsRequest,
	) ([]*location.Location, error)
}

type locationZones struct {
	byLocation map[pulid.ID]string
	fallback   string
}

func readLocationZones(
	ctx context.Context,
	reader locationZoneReader,
	tenant pagination.TenantInfo,
	fallback string,
	ids []pulid.ID,
) (locationZones, error) {
	zones := locationZones{fallback: fallback}
	if reader == nil || len(ids) == 0 {
		return zones, nil
	}

	locations, err := reader.GetByIDs(ctx, repositories.GetLocationsByIDsRequest{
		TenantInfo:  tenant,
		LocationIDs: ids,
	})
	if err != nil {
		return zones, err
	}

	zones.byLocation = make(map[pulid.ID]string, len(locations))
	for _, loc := range locations {
		if loc != nil {
			zones.byLocation[loc.ID] = loc.Timezone
		}
	}

	return zones, nil
}

func (z locationZones) at(locationID pulid.ID) *time.Location {
	loc, _ := timeutils.ResolveZone(z.byLocation[locationID], z.fallback)

	return loc
}
