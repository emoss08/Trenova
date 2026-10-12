package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeLocationZones struct {
	Zones map[pulid.ID]string `json:"-"`
}

func (f FakeLocationZones) GetByIDs(
	_ context.Context,
	req locationports.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	found := make([]*location.Location, 0, len(req.LocationIDs))
	for _, id := range req.LocationIDs {
		if zone, ok := f.Zones[id]; ok {
			found = append(found, &location.Location{ID: id, Timezone: zone})
		}
	}

	return found, nil
}
