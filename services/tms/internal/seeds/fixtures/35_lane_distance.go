package fixtures

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride/distanceoverridedb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/shared/geoutils"
)

type LaneDistanceSeed struct {
}

var LaneDistance = Seeds.Add(seed.Seed{
	Name:        "lane_distance",
	Kind:        seed.Fixture,
	Description: "Seeds road distances between the development locations, so a new load prices its miles without a routing provider",
	After:       []*seed.Seed{AdminAccount, Location},
	Run:         (&LaneDistanceSeed{}).Run,
})

func (s *LaneDistanceSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	l := locationdb.Locations
	locations, err := l.AllTenants().Where(
		l.OrganizationID.Eq(org.ID),
		l.BusinessUnitID.Eq(org.BusinessUnitID),
		l.Latitude.IsNotNull(),
		l.Longitude.IsNotNull(),
	).Order(l.ID.Asc()).All(ctx, db)
	if err != nil {
		return fmt.Errorf("read the development locations: %w", err)
	}

	for _, origin := range locations {
		for _, destination := range locations {
			if origin.ID == destination.ID {
				continue
			}
			if err = s.ensure(ctx, db, laneOverride(origin, destination)); err != nil {
				return fmt.Errorf("ensure the %s to %s distance: %w",
					origin.Name, destination.Name, err)
			}
		}
	}

	return nil
}

func laneOverride(origin, destination *location.Location) *distanceoverride.DistanceOverride {
	straight := geoutils.HaversineMiles(
		*origin.Latitude, *origin.Longitude, *destination.Latitude, *destination.Longitude,
	)
	override := &distanceoverride.DistanceOverride{
		OrganizationID:        origin.OrganizationID,
		BusinessUnitID:        origin.BusinessUnitID,
		OriginLocationID:      origin.ID,
		DestinationLocationID: destination.ID,
		Distance:              math.Round(straight * geoutils.RoadCircuityFactor),
	}
	override.RouteSignature = override.BuildRouteSignature()

	return override
}

func (s *LaneDistanceSeed) ensure(
	ctx context.Context,
	db *keel.DB,
	override *distanceoverride.DistanceOverride,
) error {
	d := distanceoverridedb.DistanceOverrides
	_, err := d.CreateOrSkip(
		ctx,
		db,
		[]*distanceoverride.DistanceOverride{override},
		d.OrganizationID,
		d.BusinessUnitID,
		d.RouteSignature,
	)
	return err
}
