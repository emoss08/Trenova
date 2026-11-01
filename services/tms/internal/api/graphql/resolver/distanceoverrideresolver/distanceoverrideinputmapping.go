package distanceoverrideresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func distanceOverrideFromInput(
	input *gqlmodel.DistanceOverrideInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*distanceoverride.DistanceOverride, error) {
	originID, err := gqlid.RequiredID("originLocationId", input.OriginLocationID)
	if err != nil {
		return nil, err
	}
	destinationID, err := gqlid.RequiredID("destinationLocationId", input.DestinationLocationID)
	if err != nil {
		return nil, err
	}
	customerID, err := gqlid.OptionalScopedID("customerId", input.CustomerID)
	if err != nil {
		return nil, err
	}

	stops := make([]*distanceoverride.DistanceOverrideStop, 0, len(input.IntermediateStops))
	for i, stop := range input.IntermediateStops {
		if stop == nil {
			continue
		}
		locationID, stopErr := gqlid.RequiredID(
			fmt.Sprintf("intermediateStops[%d].locationId", i),
			stop.LocationID,
		)
		if stopErr != nil {
			return nil, stopErr
		}
		stops = append(stops, &distanceoverride.DistanceOverrideStop{LocationID: locationID})
	}

	entity := &distanceoverride.DistanceOverride{
		ID:                    id,
		OrganizationID:        authCtx.OrganizationID,
		BusinessUnitID:        authCtx.BusinessUnitID,
		OriginLocationID:      originID,
		DestinationLocationID: destinationID,
		CustomerID:            customerID,
		Distance:              input.Distance,
		IntermediateStops:     stops,
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}
