package sim

import (
	"math"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestVehicleListAndGet(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	vehicles := requireStatus(t, srv, http.MethodGet, "/fleet/vehicles", nil, http.StatusOK).list(t)
	if len(vehicles) != 12 {
		t.Fatalf("expected only the 12 tractors, got %d", len(vehicles))
	}
	truck := requireRecord(t, vehicles, fixtureTruck1001)
	if nestedString(truck, "gateway", "serial") == "" ||
		truck["serial"] != nestedString(truck, "gateway", "serial") ||
		truck["year"] != "2022" ||
		truck["vehicleRegulationMode"] != "regulated" ||
		truck["createdAtTime"] == nil ||
		truck["updatedAtTime"] == nil {
		t.Fatalf("unexpected vehicle list item %v", truck)
	}
	if nestedString(truck, fieldExternalIDs, externalIDVinKey) != "1FUJGLDR5CLBP1001" ||
		nestedString(
			truck,
			fieldExternalIDs,
			externalIDSerialKey,
		) != nestedString(
			truck,
			"gateway",
			"serial",
		) ||
		nestedString(truck, fieldExternalIDs, "tmsVehicleId") != "unit-1001" {
		t.Fatalf("expected user and samsara.* external IDs, got %v", truck[fieldExternalIDs])
	}
	if nestedString(truck, "staticAssignedDriver", "id") != fixtureDriverAlex {
		t.Fatalf("expected the live driver pairing, got %v", truck["staticAssignedDriver"])
	}

	for _, ref := range []string{
		fixtureTruck1001,
		"samsara.vin:1FUJGLDR5CLBP1001",
		"samsara.serial:" + nestedString(truck, "gateway", "serial"),
		"tmsVehicleId:unit-1001",
	} {
		vehicle := requireStatus(
			t,
			srv,
			http.MethodGet,
			"/fleet/vehicles/"+ref,
			nil,
			http.StatusOK,
		).data(t)
		if recordID(vehicle) != fixtureTruck1001 {
			t.Fatalf("expected %s to resolve Truck 1001, got %s", ref, recordID(vehicle))
		}
		if _, has := vehicle["createdAtTime"]; has {
			t.Fatal(
				"expected the single-vehicle object to follow the Vehicle schema without timestamps",
			)
		}
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/"+fixtureTrailer2042,
		nil,
	).expectError(t, http.StatusNotFound, "vehicle")

	byDate := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/fleet/vehicles",
			map[string]string{"attributes": "Next PM Due:range(2026-04-01,2026-04-30)"},
		),
		nil,
		http.StatusOK,
	).list(t)
	for _, vehicle := range byDate {
		due := ""
		for _, raw := range vehicle["attributes"].([]any) {
			attribute := Record(raw.(map[string]any))
			if stringValue(attribute, "name") == "Next PM Due" {
				due = stringListValues(attribute["dateValues"])[0]
			}
		}
		if due < "2026-04-01" || due > "2026-04-30" {
			t.Fatalf("expected PM due in April, got %s for %s", due, recordID(vehicle))
		}
	}
	if len(byDate) == 0 {
		t.Fatal("expected date range filtering to match fixture vehicles")
	}
	created := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles?createdAfterTime=2026-01-02T00:00:00Z",
		nil,
		http.StatusOK,
	).list(t)
	if len(created) != 0 {
		t.Fatalf("expected no vehicles created after Jan 2, got %v", listIDs(created))
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles?limit=513",
		nil,
	).expectError(t, http.StatusBadRequest, "limit")
}

func TestVehiclePatchValidationAndEffects(t *testing.T) {
	t.Parallel()

	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true, WebhookURL: sink.url})
	tests := []struct {
		body     map[string]any
		fragment string
	}{
		{body: map[string]any{"vin": "SHORT"}, fragment: "vin"},
		{body: map[string]any{"vin": "1FUJGLDR5CLBP-002"}, fragment: "vin"},
		{body: map[string]any{"vin": "4V4NC9EH6EN171002"}, fragment: "already assigned"},
		{body: map[string]any{"licensePlate": "TX-1001-LONGER"}, fragment: "12"},
		{body: map[string]any{"notes": strings.Repeat("n", 256)}, fragment: "255"},
		{body: map[string]any{"auxInputType1": "jetpack"}, fragment: "auxInputType1"},
		{
			body:     map[string]any{"harshAccelerationSettingType": "rocket"},
			fragment: "harshAccelerationSettingType",
		},
		{
			body:     map[string]any{"vehicleRegulationMode": "sometimes"},
			fragment: "vehicleRegulationMode",
		},
		{body: map[string]any{"gatewaySerial": "nope"}, fragment: "XXXX-XXX-XXX"},
		{
			body: map[string]any{
				"grossVehicleWeight": map[string]any{"unit": "stone", "weight": float64(10)},
			},
			fragment: "unit",
		},
		{body: map[string]any{"odometerMeters": float64(-1)}, fragment: "odometerMeters"},
		{body: map[string]any{"staticAssignedDriverId": "404"}, fragment: "driver"},
		{body: map[string]any{"tagIds": []any{"404"}}, fragment: "tag"},
		{
			body:     map[string]any{"externalIds": map[string]any{"tmsVehicleId": "unit-1002"}},
			fragment: "already assigned",
		},
	}
	for _, testCase := range tests {
		callAPI(t, srv, http.MethodPatch, "/fleet/vehicles/"+fixtureTruck1001, testCase.body).
			expectError(t, http.StatusBadRequest, testCase.fragment)
	}
	callAPI(t, srv, http.MethodPatch, "/fleet/vehicles/404", map[string]any{"name": "x"}).
		expectError(t, http.StatusNotFound, "vehicle")

	srv.clock.SetTime(fleetTestTime.Add(time.Minute))
	patched := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/vehicles/tmsVehicleId:unit-1001",
		map[string]any{
			"name":                   "Truck 1001A",
			"odometerMeters":         float64(512_000_000),
			"engineHours":            float64(20_000),
			"auxInputType2":          "door",
			"grossVehicleWeight":     map[string]any{"unit": "kg", "weight": float64(36_000)},
			"staticAssignedDriverId": "workerId:worker-1012",
			"tagIds":                 []any{tagTractors},
			"vehicleRegulationMode":  "mixed",
		},
		http.StatusOK,
	).data(t)
	if patched["name"] != "Truck 1001A" || patched["auxInputType2"] != "door" ||
		patched["vehicleRegulationMode"] != "mixed" ||
		nestedString(patched, "staticAssignedDriver", "id") != fixtureDriverCameron ||
		!slices.Equal(tagIDsOf(t, patched), []string{tagTractors}) {
		t.Fatalf("unexpected patched vehicle %v", patched)
	}
	listItem := requireRecord(
		t,
		requireStatus(t, srv, http.MethodGet, "/fleet/vehicles", nil, http.StatusOK).list(t),
		fixtureTruck1001,
	)
	if listItem["vehicleRegulationMode"] != "unregulated" ||
		floatFromAny(listItem["vehicleWeightInKilograms"]) != 36_000 ||
		floatFromAny(listItem["vehicleWeightInPounds"]) != 79_366 {
		t.Fatalf("expected list-schema regulation mode and weights, got %v", listItem)
	}
	stats := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=gpsOdometerMeters,syntheticEngineSeconds,auxInput2&vehicleIds="+fixtureTruck1001,
		nil,
		http.StatusOK,
	).list(t)
	if floatFromAny(nestedAny(stats[0], "gpsOdometerMeters", "value")) != 512_000_000 ||
		floatFromAny(nestedAny(stats[0], "syntheticEngineSeconds", "value")) != 72_000_000 ||
		nestedString(stats[0], "auxInput2", "name") != "Door" {
		t.Fatalf(
			"expected manual odometer, engine hours and aux input to flow into stats, got %v",
			stats[0],
		)
	}
	driver := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/"+fixtureDriverCameron,
		nil,
		http.StatusOK,
	).data(t)
	if nestedString(driver, "staticAssignedVehicle", "id") != fixtureTruck1001 {
		t.Fatalf(
			"expected the static assignment on the driver, got %v",
			driver["staticAssignedVehicle"],
		)
	}
	events := waitForWebhookEvents(t, sink, eventVehicleUpdated, 1)
	vehicle, ok := anyAsMap(webhookData(t, events[0])["vehicle"])
	if !ok || stringValue(Record(vehicle), "id") != fixtureTruck1001 {
		t.Fatalf("expected a {vehicle} VehicleUpdated payload, got %v", events[0].Data)
	}
}

func TestVehicleLocationsAgreeWithStats(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	locations := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations",
		nil,
		http.StatusOK,
	).list(t)
	if len(locations) != 12 {
		t.Fatalf("expected a location per tractor, got %d", len(locations))
	}
	stats := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=gps",
		nil,
		http.StatusOK,
	).list(t)
	for _, location := range locations {
		stat := requireRecord(t, stats, recordID(location))
		if floatFromAny(
			nestedAny(location, "location", "latitude"),
		) != floatFromAny(
			nestedAny(stat, "gps", "latitude"),
		) ||
			floatFromAny(
				nestedAny(location, "location", "speed"),
			) != floatFromAny(
				nestedAny(stat, "gps", "speedMilesPerHour"),
			) {
			t.Fatalf(
				"expected locations and stats to agree for %s: %v vs %v",
				recordID(location),
				location["location"],
				stat["gps"],
			)
		}
		if nestedString(location, "location", "reverseGeo", "formattedLocation") == "" {
			t.Fatalf("expected reverse geocoding on %s", recordID(location))
		}
	}
	past := fleetTestTime.Add(-time.Hour).Format(time.RFC3339)
	atTime := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations?time="+past+"&vehicleIds="+fixtureTruck1001,
		nil,
		http.StatusOK,
	).list(t)
	if nestedString(atTime[0], "location", "time") != past {
		t.Fatalf("expected the time parameter to select the sample, got %v", atTime[0]["location"])
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations?time=soon",
		nil,
	).expectError(t, http.StatusBadRequest, "time")

	history := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/vehicles/locations/history", map[string]string{
			"startTime":  fleetTestTime.Add(-20 * time.Minute).Format(time.RFC3339),
			"endTime":    fleetTestTime.Format(time.RFC3339),
			"vehicleIds": fixtureTruck1001,
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(history) != 1 || len(history[0]["locations"].([]any)) < 10 {
		t.Fatalf("expected a 2-minute location history, got %v", history)
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations/history",
		nil,
	).expectError(t, http.StatusBadRequest, "startTime")

	first := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations/feed",
		nil,
		http.StatusOK,
	)
	cursor := stringValue(first.pagination(t), "endCursor")
	if cursor == "" || len(first.list(t)) != 12 {
		t.Fatalf("expected the latest location per vehicle and a cursor, got %s", first.Body)
	}
	srv.clock.Step(6 * time.Minute)
	next := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/locations/feed?after="+cursor,
		nil,
		http.StatusOK,
	)
	for _, record := range next.list(t) {
		if len(record["locations"].([]any)) != 3 {
			t.Fatalf(
				"expected three 2-minute samples after 6 minutes, got %d",
				len(record["locations"].([]any)),
			)
		}
	}
}

func TestVehicleImmobilizerStream(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/immobilizer/stream?startTime=2026-01-01T00:00:00Z",
		nil,
	).
		expectError(t, http.StatusBadRequest, "vehicleIds")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/immobilizer/stream?vehicleIds="+fixtureTruck1001,
		nil,
	).
		expectError(t, http.StatusBadRequest, "startTime")
	records := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/vehicles/immobilizer/stream", map[string]string{
			"vehicleIds": fixtureTruck1001 + ",tmsVehicleId:unit-1002,samsara.vin:3AKJHHDR5JSKA1005",
			"startTime":  "2026-01-01T00:00:00Z",
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(records) == 0 {
		t.Fatal("expected immobilizer state records")
	}
	seen := map[string]bool{}
	previous := ""
	for _, record := range records {
		seen[stringValue(record, "vehicleId")] = true
		happened := stringValue(record, "happenedAtTime")
		if happened < previous {
			t.Fatal("expected records ordered by happenedAtTime")
		}
		previous = happened
		relays := record["relayStates"].([]any)
		if len(relays) != 2 || stringValue(Record(relays[0].(map[string]any)), "id") != relayOne {
			t.Fatalf("unexpected relay states %v", relays)
		}
	}
	if !seen[fixtureTruck1001] || !seen[fixtureTruck1005] || seen[fixtureTruck1002] {
		t.Fatalf("expected states only for vehicles with immobilizers, got %v", seen)
	}
	stat := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=engineImmobilizer",
		nil,
		http.StatusOK,
	).list(t)
	withImmobilizer := 0
	for _, row := range stat {
		if immobilizer, ok := anyAsMap(row["engineImmobilizer"]); ok {
			withImmobilizer++
			if stringValue(Record(immobilizer), "state") != ignitionEnabled {
				t.Fatalf("unexpected immobilizer stat %v", immobilizer)
			}
		}
	}
	if withImmobilizer != 3 {
		t.Fatalf(
			"expected the three fitted trucks to report the immobilizer, got %d",
			withImmobilizer,
		)
	}
}

func TestGazetteerReverseGeocode(t *testing.T) {
	t.Parallel()

	if got := nearestPlaceDescription(30.2672, -97.7431); got != "Austin, TX" {
		t.Fatalf("expected the city name inside the city, got %q", got)
	}
	got := nearestPlaceDescription(30.40, -97.74)
	if !strings.HasSuffix(got, ", TX") || !strings.Contains(got, " mi ") {
		t.Fatalf("expected a distance and direction description, got %q", got)
	}
	if nearestPlaceDescription(45.0, -120.0) != "" {
		t.Fatal("expected no description far outside the gazetteer")
	}
	if compassPoint(0) != "N" || compassPoint(91) != "E" || compassPoint(359) != "N" ||
		math.IsNaN(radians(1)) {
		t.Fatal("unexpected compass points")
	}
}
