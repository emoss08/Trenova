package shipmentpreview

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type LocationZoneReader interface {
	GetByIDs(
		ctx context.Context,
		req locationports.GetLocationsByIDsRequest,
	) ([]*location.Location, error)
}

type LocationZones struct {
	byLocation map[pulid.ID]string
	Fallback   string `json:"-"`
}

func ReadLocationZones(
	ctx context.Context,
	reader LocationZoneReader,
	tenant pagination.TenantInfo,
	fallback string,
	ids []pulid.ID,
) (LocationZones, error) {
	zones := LocationZones{Fallback: fallback}
	if reader == nil || len(ids) == 0 {
		return zones, nil
	}

	locations, err := reader.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
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

func (z LocationZones) At(locationID pulid.ID) *time.Location {
	loc, _ := timeutils.ResolveZone(z.byLocation[locationID], z.Fallback)

	return loc
}
