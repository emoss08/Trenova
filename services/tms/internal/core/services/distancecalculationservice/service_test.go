package distancecalculationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestRouteOptionsFromDistanceProfile(t *testing.T) {
	t.Parallel()

	profile := distanceprofile.NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	profile.DataVersion = "Current"
	profile.RoutingType = "Shortest"
	profile.DistanceUnits = "Kilometers"
	profile.LocationGranularity = "coordinates"
	profile.HighwayOnly = true
	profile.TollRoads = false
	profile.BordersOpen = false
	profile.IncludeTollData = true
	options := profile.RouteOptions()

	assert.Equal(t, "Current", options.DataVersion)
	assert.Equal(t, "NA", options.Region)
	assert.Equal(t, "Shortest", options.RoutingType)
	assert.Equal(t, "Kilometers", options.DistanceUnits)
	assert.Equal(t, "coordinates", options.LocationGranularity)
	assert.True(t, options.HighwayOnly)
	assert.False(t, options.TollRoads)
	assert.False(t, options.BordersOpen)
	assert.Equal(t, "Truck", options.VehicleType)
	assert.Empty(t, options.Hazmat)
	assert.True(t, options.IncludeTollData)
}

func TestOptionsGranularity(t *testing.T) {
	t.Parallel()

	options := distanceprofile.NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_")).RouteOptions()
	options.LocationGranularity = "coordinates"
	assert.Equal(t, "Coordinates", optionsGranularity(options))
}

func TestCanResolveMoveDistance(t *testing.T) {
	t.Parallel()

	assert.False(t, canResolveMoveDistance(&shipment.ShipmentMove{}))
	assert.False(t, canResolveMoveDistance(&shipment.ShipmentMove{
		Stops: []*shipment.Stop{{LocationID: pulid.MustNew("loc_")}},
	}))
	assert.True(t, canResolveMoveDistance(&shipment.ShipmentMove{
		Stops: []*shipment.Stop{
			{LocationID: pulid.MustNew("loc_")},
			{LocationID: pulid.MustNew("loc_")},
		},
	}))
}

func TestHazmatTypesForShipment(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{
		Commodities: []*shipment.ShipmentCommodity{
			{
				Commodity: &commodity.Commodity{
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						Class: hazardousmaterial.HazardousClass3,
					},
				},
			},
			{
				Commodity: &commodity.Commodity{
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						Class:            hazardousmaterial.HazardousClass8,
						InhalationHazard: true,
						MarinePollutant:  true,
					},
				},
			},
			{
				Commodity: &commodity.Commodity{
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						Class: hazardousmaterial.HazardousClass3,
					},
				},
			},
		},
	}

	assert.Equal(t, []string{"Caustic", "Flammable", "HarmfulToWater", "Inhalants"}, hazmatTypesForShipment(entity))
}
