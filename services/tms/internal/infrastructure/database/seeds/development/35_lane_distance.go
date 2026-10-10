package development

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/uptrace/bun"
)

type LaneDistanceSeed struct {
	seedhelpers.BaseSeed
}

func NewLaneDistanceSeed() *LaneDistanceSeed {
	seed := &LaneDistanceSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"LaneDistance",
		"1.0.0",
		"Seeds road distances between the development locations, so a new load prices its miles without a routing provider",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations, seedhelpers.SeedLocation)

	return seed
}

func (s *LaneDistanceSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			locations := make([]*location.Location, 0, 16)
			cols := buncolgen.LocationColumns
			if err = tx.NewSelect().
				Model(&locations).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Where(cols.Latitude.IsNotNull()).
				Where(cols.Longitude.IsNotNull()).
				Scan(ctx); err != nil {
				return fmt.Errorf("read the development locations: %w", err)
			}

			for _, origin := range locations {
				for _, destination := range locations {
					if origin.ID == destination.ID {
						continue
					}
					if err = s.ensure(ctx, tx, laneOverride(origin, destination)); err != nil {
						return fmt.Errorf("ensure the %s to %s distance: %w",
							origin.Name, destination.Name, err)
					}
				}
			}

			return nil
		},
	)
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
	tx bun.Tx,
	override *distanceoverride.DistanceOverride,
) error {
	cols := buncolgen.DistanceOverrideColumns
	exists, err := tx.NewSelect().
		Model((*distanceoverride.DistanceOverride)(nil)).
		Where(cols.OrganizationID.Eq(), override.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), override.BusinessUnitID).
		Where(cols.RouteSignature.Eq(), override.RouteSignature).
		Exists(ctx)
	if err != nil || exists {
		return err
	}

	_, err = tx.NewInsert().Model(override).Exec(ctx)

	return err
}
