package pcmiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func float64Ptr(value float64) *float64 { return &value }

func TestBuildStopPrefersTrimblePlaceID(t *testing.T) {
	t.Parallel()

	stop := buildStop(Stop{
		TrimblePlaceID: "place_123",
		Latitude:       float64Ptr(40.0),
		Longitude:      float64Ptr(-75.0),
		City:           "Philadelphia",
	}, "NA", 0, 3)

	require.Equal(t, "place_123", stop["PlaceId"])
	require.NotContains(t, stop, "Coords")
	require.NotContains(t, stop, "Address")
	require.Equal(t, "Origin", stop["Label"])
}

func TestBuildStopFallsBackToCoordinates(t *testing.T) {
	t.Parallel()

	stop := buildStop(Stop{
		Latitude:  float64Ptr(40.0),
		Longitude: float64Ptr(-75.0),
		City:      "Philadelphia",
	}, "NA", 1, 3)

	require.Equal(t, map[string]any{"Lat": 40.0, "Lon": -75.0}, stop["Coords"])
	require.NotContains(t, stop, "Address")
	require.Equal(t, "Stop 1", stop["Label"])
}

func TestBuildStopFallsBackToAddress(t *testing.T) {
	t.Parallel()

	stop := buildStop(Stop{
		AddressLine: "123 Main St",
		City:        "Philadelphia",
		State:       "PA",
		PostalCode:  "19103",
	}, "NA", 2, 3)

	require.Equal(t, map[string]any{
		"Country":       "US",
		"StreetAddress": "123 Main St",
		"City":          "Philadelphia",
		"State":         "PA",
		"Zip":           "19103",
	}, stop["Address"])
	require.Equal(t, "Destination", stop["Label"])
}

func TestVehicleTypeCode(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, vehicleTypeCode(""))
	require.Equal(t, 0, vehicleTypeCode("truck"))
	require.Equal(t, 2, vehicleTypeCode("Auto"))
	require.Equal(t, 1, vehicleTypeCode("LightTruck"))
	require.Equal(t, 1, vehicleTypeCode("light truck"))
}

func TestRoutingTypeCode(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, routingTypeCode(""))
	require.Equal(t, 0, routingTypeCode("practical"))
	require.Equal(t, 1, routingTypeCode("Shortest"))
	require.Equal(t, 2, routingTypeCode("fastest"))
}

func TestDistanceUnitsCode(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, distanceUnitsCode(""))
	require.Equal(t, 0, distanceUnitsCode("miles"))
	require.Equal(t, 1, distanceUnitsCode("Kilometers"))
	require.Equal(t, 1, distanceUnitsCode("kilometres"))
	require.Equal(t, 1, distanceUnitsCode("km"))
}

func TestTollRoadsCode(t *testing.T) {
	t.Parallel()

	require.Equal(t, 3, tollRoadsCode(true))
	require.Equal(t, 2, tollRoadsCode(false))
}

func TestRegionCode(t *testing.T) {
	t.Parallel()

	require.Equal(t, 1, regionCode("AF"))
	require.Equal(t, 2, regionCode("as"))
	require.Equal(t, 3, regionCode("EU"))
	require.Equal(t, 5, regionCode("OC"))
	require.Equal(t, 6, regionCode("SA"))
	require.Equal(t, 7, regionCode("ME"))
	require.Equal(t, 4, regionCode("NA"))
	require.Equal(t, 4, regionCode(""))
}

func TestHazmatCodesSkipsUnknownValues(t *testing.T) {
	t.Parallel()

	codes := hazmatCodes([]string{
		"General", "caustic", "explosives", "flammable",
		"inhalants", "radioactive", "harmful to water", "tunnel", "unknown",
	})

	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, codes)
}

func TestBuildOptionsOmitsZeroDimensions(t *testing.T) {
	t.Parallel()

	values := buildOptions(RouteOptions{TollRoads: true, BordersOpen: true})

	require.Equal(t, 3, values["TollRoads"])
	require.Equal(t, true, values["BordersOpen"])
	require.Equal(t, 0, values["DistanceUnits"])
	absentKeys := []string{
		"Height", "Length", "Width", "Weight",
		"Axles", "HazMatTypes", "ProfileName", "HighwayOnly",
	}
	for _, key := range absentKeys {
		require.NotContains(t, values, key)
	}
}

func TestBuildOptionsIncludesConfiguredDimensions(t *testing.T) {
	t.Parallel()

	values := buildOptions(RouteOptions{
		ProfileName:   " TractorTrailer ",
		HighwayOnly:   true,
		VehicleHeight: 13.5,
		VehicleLength: 53,
		VehicleWidth:  8.5,
		VehicleWeight: 80000,
		Axles:         5,
		Hazmat:        []string{"general"},
		DistanceUnits: "km",
	})

	require.Equal(t, "TractorTrailer", values["ProfileName"])
	require.Equal(t, true, values["HighwayOnly"])
	require.Equal(t, 13.5, values["Height"])
	require.Equal(t, 53.0, values["Length"])
	require.Equal(t, 8.5, values["Width"])
	require.Equal(t, 80000.0, values["Weight"])
	require.Equal(t, 5, values["Axles"])
	require.Equal(t, []int{1}, values["HazMatTypes"])
	require.Equal(t, 1, values["DistanceUnits"])
}

func TestParseMileageResponseFiltersNonMileageReports(t *testing.T) {
	t.Parallel()

	results := parseMileageResponse([]routeReport{
		{
			Type:    "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
			RouteID: "route-1",
			ReportLines: []reportLine{
				{TMiles: "120.5"},
			},
		},
		{
			Type:    "TollReport:http://pcmiler.alk.com/APIs/v1.0",
			RouteID: "route-1",
		},
	})

	require.Len(t, results, 1)
	require.Equal(t, "route-1", results[0].RouteID)
	require.InDelta(t, 120.5, results[0].Distance, 0.0001)
}

func TestParseMileageResponseCollectsWarningsAndLastDistance(t *testing.T) {
	t.Parallel()

	results := parseMileageResponse([]routeReport{
		{
			Type:    "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
			RouteID: "route-2",
			ReportLines: []reportLine{
				{TMiles: "50.0", Warn: "toll data unavailable"},
				{TMiles: "", Dist: "0", LMiles: "not-a-number"},
				{TMiles: "75.25"},
			},
		},
	})

	require.Len(t, results, 1)
	require.InDelta(t, 75.25, results[0].Distance, 0.0001)
	require.Equal(t, []string{"toll data unavailable"}, results[0].Warnings)
}

func TestLastParsedPositiveIgnoresZeroAndGarbage(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 10.0, lastParsedPositive(10.0, "0", "-5", "abc", ""), 0.0001)
	require.InDelta(t, 42.5, lastParsedPositive(10.0, "12", "42.5"), 0.0001)
}

func TestBuildRouteReportsPayloadShape(t *testing.T) {
	t.Parallel()

	payload := buildRouteReportsPayload([]RouteRequest{
		{
			RouteID: "route-1",
			Stops: []Stop{
				{City: "Philadelphia", State: "PA"},
				{City: "Pittsburgh", State: "PA"},
			},
			Options: RouteOptions{
				Region:      "NA",
				VehicleType: "auto",
				RoutingType: "shortest",
			},
		},
	})

	routes, ok := payload["ReportRoutes"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, routes, 1)

	route := routes[0]
	require.Equal(t, "route-1", route["RouteId"])
	require.Equal(t, 2, route["VehicleType"])
	require.Equal(t, 1, route["RoutingType"])

	stops, ok := route["Stops"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, stops, 2)
	require.Equal(t, "Origin", stops[0]["Label"])
	require.Equal(t, "Destination", stops[1]["Label"])

	reportTypes, ok := route["ReportTypes"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, reportTypes, 1)
	require.Equal(
		t,
		"MileageReportType:http://pcmiler.alk.com/APIs/v1.0",
		reportTypes[0]["__type"],
	)
}
