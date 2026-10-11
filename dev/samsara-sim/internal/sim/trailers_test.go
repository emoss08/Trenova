package sim

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTrailerFixtureAndCRUD(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	trailers := requireStatus(t, srv, http.MethodGet, "/fleet/trailers", nil, http.StatusOK).list(t)
	if len(trailers) != 14 {
		t.Fatalf("expected 14 fixture trailers, got %d", len(trailers))
	}
	names := make([]string, 0, len(trailers))
	for _, trailer := range trailers {
		names = append(names, stringValue(trailer, "name"))
		if _, has := trailer["attributes"]; has {
			t.Fatal("expected list items to follow the list schema without attributes")
		}
		if nestedString(trailer, "installedGateway", "serial") == "" ||
			len(stringValue(trailer, "trailerSerialNumber")) != 17 {
			t.Fatalf("expected gateway and 17-character serial on %v", trailer)
		}
	}
	if !slices.Contains(names, "Trailer 2042") || !slices.Contains(names, "Trailer 2054") {
		t.Fatalf("expected Trailer 2041-2054, got %v", names)
	}

	single := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/tmsTrailerId:trailer-2042",
		nil,
		http.StatusOK,
	).data(t)
	if recordID(single) != fixtureTrailer2042 || len(single["attributes"].([]any)) == 0 {
		t.Fatalf("expected the single trailer schema with attributes, got %v", single)
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/"+fixtureTruck1001,
		nil,
	).expectError(t, http.StatusNotFound, "trailer")

	for _, testCase := range []struct {
		body     map[string]any
		fragment string
	}{
		{body: map[string]any{}, fragment: "name"},
		{body: map[string]any{"name": "T", "licensePlate": strings.Repeat("P", 13)}, fragment: "12"},
		{body: map[string]any{"name": "T", "notes": strings.Repeat("n", 256)}, fragment: "255"},
		{body: map[string]any{"name": "T", "externalIds": map[string]any{"tmsTrailerId": "trailer-2042"}}, fragment: "already assigned"},
		{body: map[string]any{"name": "T", "tagIds": []any{"1"}}, fragment: "tag"},
	} {
		callAPI(
			t,
			srv,
			http.MethodPost,
			"/fleet/trailers",
			testCase.body,
		).expectError(t, http.StatusBadRequest, testCase.fragment)
	}
	created := requireStatus(t, srv, http.MethodPost, "/fleet/trailers", map[string]any{
		"name":                "Trailer 3001",
		"licensePlate":        "TX-T3001",
		"trailerSerialNumber": "1UYVS2530PU999001",
		"enabledForMobile":    true,
		"tagIds":              []any{tagDryVanTrailers},
		"externalIds":         map[string]any{"tmsTrailerId": "trailer-3001"},
	}, http.StatusOK).data(t)
	trailerID := recordID(created)
	if created["enabledForMobile"] != true ||
		!slices.Equal(tagIDsOf(t, created), []string{tagDryVanTrailers}) {
		t.Fatalf("unexpected created trailer %v", created)
	}
	asset := requireRecord(
		t,
		requireStatus(t, srv, http.MethodGet, "/assets?type=trailer", nil, http.StatusOK).list(t),
		trailerID,
	)
	if stringValue(asset, "serialNumber") != "1UYVS2530PU999001" {
		t.Fatalf("expected the trailer in /assets with its serial number, got %v", asset)
	}

	patched := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/trailers/tmsTrailerId:trailer-3001",
		map[string]any{
			"notes":          "Spotted at Austin",
			"odometerMeters": float64(100_000_000),
			"licensePlate":   nil,
		},
		http.StatusOK,
	).data(t)
	if patched["notes"] != "Spotted at Austin" || patched["licensePlate"] != nil {
		t.Fatalf("unexpected patched trailer %v", patched)
	}
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/fleet/trailers/"+fixtureTruck1001,
		map[string]any{"name": "x"},
	).
		expectError(t, http.StatusNotFound, "trailer")
	callAPI(t, srv, http.MethodDelete, "/fleet/trailers/tmsTrailerId:trailer-3001", nil).
		expectError(t, http.StatusNotFound, "trailer")
	requireStatus(
		t,
		srv,
		http.MethodDelete,
		"/fleet/trailers/"+trailerID,
		nil,
		http.StatusNoContent,
	)
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/"+trailerID,
		nil,
	).expectError(t, http.StatusNotFound, "trailer")
	tag := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/"+tagDryVanTrailers,
		nil,
		http.StatusOK,
	).data(t)
	if slices.Contains(listIDs(recordsFromAny(tag["assets"])), trailerID) {
		t.Fatal("expected deletion to remove the trailer from its tags")
	}
}

func recordsFromAny(raw any) []Record {
	items, _ := raw.([]any)
	out := make([]Record, 0, len(items))
	for _, item := range items {
		if mapped, ok := anyAsMap(item); ok {
			out = append(out, Record(mapped))
		}
	}
	return out
}

func TestTrailerFollowsTractorAndParkedTrailersStay(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	trailerStats := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats?types=gps&trailerIds="+fixtureTrailer2042+","+fixtureTrailer2053,
		nil,
		http.StatusOK,
	).list(t)
	tractor := requireStatus(t, srv, http.MethodGet,
		"/fleet/vehicles/stats?types=gps&vehicleIds="+fixtureTruck1002, nil, http.StatusOK).list(t)[0]
	coupled := requireRecord(t, trailerStats, fixtureTrailer2042)
	distance := haversineMeters(
		floatFromAny(
			nestedAny(coupled, "gps", "latitude"),
		),
		floatFromAny(nestedAny(coupled, "gps", "longitude")),
		floatFromAny(
			nestedAny(tractor, "gps", "latitude"),
		),
		floatFromAny(nestedAny(tractor, "gps", "longitude")),
	)
	if distance < 5 || distance > 20 {
		t.Fatalf("expected Trailer 2042 a hitch length behind Truck 1002, got %.1fm", distance)
	}
	if _, isInt := nestedAny(coupled, "gps", "speedMilesPerHour").(float64); !isInt {
		t.Fatalf(
			"expected numeric trailer speed, got %T",
			nestedAny(coupled, "gps", "speedMilesPerHour"),
		)
	}
	parked := requireRecord(t, trailerStats, fixtureTrailer2053)
	if nestedString(
		parked,
		"gps",
		"reverseGeo",
		"formattedLocation",
	) != "100 Fleet Ave, Austin, TX 78701" ||
		floatFromAny(nestedAny(parked, "gps", "speedMilesPerHour")) != 0 {
		t.Fatalf("expected Trailer 2053 parked at the Austin yard, got %v", parked["gps"])
	}
	srv.clock.Step(30 * time.Minute)
	later := requireStatus(t, srv, http.MethodGet,
		"/fleet/trailers/stats?types=gps&trailerIds="+fixtureTrailer2053, nil, http.StatusOK).list(t)[0]
	if floatFromAny(
		nestedAny(later, "gps", "latitude"),
	) != floatFromAny(
		nestedAny(parked, "gps", "latitude"),
	) {
		t.Fatal("expected a parked trailer not to move")
	}

	stream := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/assets/location-and-speed/stream", map[string]string{
			"ids":       fixtureTrailer2042,
			"startTime": fleetTestTime.Add(-10 * time.Minute).Format(time.RFC3339),
			"endTime":   fleetTestTime.Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(stream) == 0 || nestedString(stream[0], "asset", "id") != fixtureTrailer2042 {
		t.Fatalf("expected trailers in the asset location stream, got %v", stream)
	}
}

func TestTrailerStatsTypesAndReeferData(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats",
		nil,
	).expectError(t, http.StatusBadRequest, "types")
	callAPI(t, srv, http.MethodGet, "/fleet/trailers/stats?types=gps,fuelPercents", nil).
		expectError(t, http.StatusBadRequest, "fuelPercents")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats?types=gps,reeferStateZone1,reeferRunMode,reeferAlarms",
		nil,
	).
		expectError(t, http.StatusBadRequest, "up to 3")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats/feed?types=gps&decorations=reeferRunMode,reeferAlarms,gpsOdometerMeters",
		nil,
	).
		expectError(t, http.StatusBadRequest, "up to 2")

	rows := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats?types=reeferSupplyAirTemperatureMilliCZone1,reeferSetPointTemperatureMilliCZone2,carrierReeferState",
		nil,
		http.StatusOK,
	).list(t)
	if len(rows) != 14 {
		t.Fatalf("expected every trailer in the snapshot, got %d", len(rows))
	}
	for _, row := range rows {
		_, hasSupply := row["reeferSupplyAirTemperatureMilliCZone1"]
		_, hasZone2 := row["reeferSetPointTemperatureMilliCZone2"]
		_, hasCarrier := row["carrierReeferState"]
		switch recordID(row) {
		case fixtureTrailer2049:
			if !hasSupply || !hasZone2 || !hasCarrier {
				t.Fatalf("expected the multi-zone Carrier reefer to report all three, got %v", row)
			}
		case fixtureTrailer2042, "281474979348663", fixtureTrailer2053:
			if !hasSupply || hasZone2 || hasCarrier {
				t.Fatalf("expected a single-zone Thermo King reefer, got %v", row)
			}
			supply := floatFromAny(nestedAny(row, "reeferSupplyAirTemperatureMilliCZone1", "value"))
			if supply < -30_000 || supply > 15_000 {
				t.Fatalf("unexpected reefer supply temperature %v", supply)
			}
		default:
			if hasSupply || hasZone2 || hasCarrier {
				t.Fatalf("expected dry vans to report no reefer data, got %v", row)
			}
		}
	}

	feed := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/trailers/stats/feed?types=reeferStateZone1,gpsOdometerMeters&decorations=gps",
		nil,
		http.StatusOK,
	)
	feedRows := feed.list(t)
	if len(feedRows) != 14 {
		t.Fatalf("expected every trailer on the feed page, got %d", len(feedRows))
	}
	for _, row := range feedRows {
		states, ok := row["reeferStateZone1"].([]any)
		if !ok {
			t.Fatalf("expected an array even for trailers without data, got %v", row)
		}
		if recordID(row) == fixtureTrailer2042 {
			if len(states) != 1 ||
				nestedString(
					Record(states[0].(map[string]any)),
					fieldDecorations,
					"gps",
					"reverseGeo",
					"formattedLocation",
				) == "" {
				t.Fatalf("expected a decorated reefer state, got %v", states)
			}
		}
	}
	history := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/trailers/stats/history", map[string]string{
			"types":      "reeferReturnAirTemperatureMilliCZone1,reeferDoorStateZone1",
			"trailerIds": "tmsTrailerId:trailer-2042",
			"startTime":  fleetTestTime.Add(-time.Hour).Format(time.RFC3339),
			"endTime":    fleetTestTime.Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(history) != 1 || len(history[0]["reeferReturnAirTemperatureMilliCZone1"].([]any)) != 31 {
		t.Fatalf("expected an hour of 2-minute reefer samples, got %v", history)
	}
	for _, raw := range history[0]["reeferDoorStateZone1"].([]any) {
		if value := stringValue(
			Record(raw.(map[string]any)),
			"value",
		); value != "open" &&
			value != "closed" {
			t.Fatalf("unexpected door state %q", value)
		}
	}
}
