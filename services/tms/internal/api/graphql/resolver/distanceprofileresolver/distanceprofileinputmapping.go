package distanceprofileresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/authctx"
)

func distanceProfileFromInput(
	input *gqlmodel.DistanceProfileInput,
	authCtx *authctx.AuthContext,
) *distanceprofile.DistanceProfile {
	entity := &distanceprofile.DistanceProfile{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyDistanceProfileInput(entity, input)

	return entity
}

func applyDistanceProfileInput(
	entity *distanceprofile.DistanceProfile,
	input *gqlmodel.DistanceProfileInput,
) {
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Status = input.Status
	entity.IsDefault = input.IsDefault
	entity.Provider = integration.Type(input.Provider)
	entity.DataVersion = gqlscalar.StringValue(input.DataVersion)
	entity.Region = input.Region
	entity.RoutingType = gqlscalar.StringValue(input.RoutingType)
	entity.DistanceUnits = gqlscalar.StringValue(input.DistanceUnits)
	entity.LocationGranularity = input.LocationGranularity
	entity.ProfileName = gqlscalar.StringValue(input.ProfileName)
	entity.HighwayOnly = input.HighwayOnly
	entity.TollRoads = input.TollRoads
	entity.BordersOpen = input.BordersOpen
	entity.IncludeTollData = input.IncludeTollData
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
