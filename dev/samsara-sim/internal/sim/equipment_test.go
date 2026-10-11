package sim

import (
	"net/http"
	"testing"
	"time"
)

func TestEquipmentListGetAndLocations(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	equipment := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment",
		nil,
		http.StatusOK,
	).list(t)
	if len(equipment) != 5 {
		t.Fatalf("expected four reefer units and a yard spotter, got %d", len(equipment))
	}
	reefer := requireRecord(t, equipment, fixtureReefer2042)
	if reefer["assetSerial"] == nil ||
		nestedString(reefer, "installedGateway", "model") != "AG26" ||
		nestedString(reefer, fieldExternalIDs, "tmsEquipmentId") == "" {
		t.Fatalf("unexpected equipment shape %v", reefer)
	}
	single := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/"+fixtureYardSpotter,
		nil,
		http.StatusOK,
	).data(t)
	if nestedString(single, "installedGateway", "model") != digitalOutputGatewayModel {
		t.Fatalf("expected the AG53 spotter, got %v", single)
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/"+fixtureTrailer2042,
		nil,
	).expectError(t, http.StatusNotFound, "equipment")

	locations := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/locations?equipmentIds="+fixtureReefer2042,
		nil,
		http.StatusOK,
	).list(t)
	trailer := requireStatus(t, srv, http.MethodGet, "/fleet/trailers/stats?types=gps&trailerIds="+fixtureTrailer2042, nil, http.StatusOK).list(t)[0]
	if floatFromAny(
		nestedAny(locations[0], "location", "latitude"),
	) != floatFromAny(
		nestedAny(trailer, "gps", "latitude"),
	) {
		t.Fatalf(
			"expected the reefer unit to ride with its trailer, got %v vs %v",
			locations[0]["location"],
			trailer["gps"],
		)
	}
	history := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/equipment/locations/history", map[string]string{
			"startTime":    fleetTestTime.Add(-10 * time.Minute).Format(time.RFC3339),
			"endTime":      fleetTestTime.Format(time.RFC3339),
			"equipmentIds": fixtureYardSpotter,
		}),
		nil,
		http.StatusOK,
	).list(t)
	samples := history[0]["locations"].([]any)
	if len(samples) != 6 {
		t.Fatalf("expected six samples, got %d", len(samples))
	}
	moving := 0
	for _, raw := range samples {
		sample := Record(raw.(map[string]any))
		if floatFromAny(sample["speed"]) > 0 {
			moving++
		}
		distance := haversineMeters(
			floatFromAny(sample["latitude"]),
			floatFromAny(sample["longitude"]),
			30.2672,
			-97.7431,
		)
		if distance > yardLoopRadiusMeters+5 {
			t.Fatalf("expected the spotter to stay in the yard, got %.0fm away", distance)
		}
	}
	if moving != len(samples) {
		t.Fatal("expected the spotter to work the yard during the day shift")
	}
	night := newFleetTestServer(
		t,
		fleetServerOptions{Dataset: true, At: time.Date(2026, time.March, 4, 9, 0, 0, 0, time.UTC)},
	)
	parked := requireStatus(t, night, http.MethodGet, "/fleet/equipment/stats?types=gps,gatewayEngineStates&equipmentIds="+fixtureYardSpotter, nil, http.StatusOK).list(t)[0]
	if floatFromAny(nestedAny(parked, "gps", "speedMilesPerHour")) != 0 ||
		nestedString(parked, "gatewayEngineState", "value") != engineStateOff {
		t.Fatalf("expected the spotter parked and off at night, got %v", parked)
	}

	feed := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/locations/feed",
		nil,
		http.StatusOK,
	)
	if len(feed.list(t)) != 5 || stringValue(feed.pagination(t), "endCursor") == "" {
		t.Fatalf("expected the latest location per unit and a cursor, got %s", feed.Body)
	}
}

func TestEquipmentStats(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/stats",
		nil,
	).expectError(t, http.StatusBadRequest, "types")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/stats?types=gps,engineRpm,fuelPercents,obdEngineStates,obdEngineSeconds",
		nil,
	).
		expectError(t, http.StatusBadRequest, "up to 4")
	callAPI(t, srv, http.MethodGet, "/fleet/equipment/stats?types=engineStates", nil).
		expectError(t, http.StatusBadRequest, "engineStates")

	rows := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/stats?types=obdEngineSeconds,gatewayJ1939EngineSeconds,fuelPercents,engineTotalIdleTimeMinutes",
		nil,
		http.StatusOK,
	).list(t)
	reefer := requireRecord(t, rows, fixtureReefer2042)
	if reefer["obdEngineSeconds"] == nil || reefer["engineSeconds"] == nil ||
		reefer["fuelPercent"] == nil || reefer["engineTotalIdleTimeMinutes"] == nil {
		t.Fatalf("expected reefer telemetry keyed by the snapshot schema, got %v", reefer)
	}
	trailer := requireStatus(t, srv, http.MethodGet,
		"/fleet/trailers/stats?types=reeferObdEngineSeconds,reeferFuelPercent&trailerIds="+fixtureTrailer2042, nil, http.StatusOK).list(t)[0]
	if floatFromAny(
		nestedAny(trailer, "reeferObdEngineSeconds", "value"),
	) != floatFromAny(
		nestedAny(reefer, "obdEngineSeconds", "value"),
	) ||
		floatFromAny(
			nestedAny(trailer, "reeferFuelPercent", "value"),
		) != floatFromAny(
			nestedAny(reefer, "fuelPercent", "value"),
		) {
		t.Fatalf(
			"expected the trailer reefer stats to match the mounted unit, got %v vs %v",
			trailer,
			reefer,
		)
	}
	spotter := requireRecord(t, rows, fixtureYardSpotter)
	if spotter["engineSeconds"] != nil {
		t.Fatal("expected J1939 engine seconds only from AG26 gateways")
	}

	feed := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/equipment/stats/feed?types=gatewayEngineStates,engineRpm",
		nil,
		http.StatusOK,
	)
	row := requireRecord(t, feed.list(t), fixtureReefer2042)
	if len(row["gatewayEngineStates"].([]any)) != 1 || len(row["engineRpm"].([]any)) != 1 {
		t.Fatalf("expected plural feed keys with the latest sample, got %v", row)
	}
	history := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/equipment/stats/history", map[string]string{
			"types":     "obdEngineStates",
			"startTime": fleetTestTime.Add(-time.Hour).Format(time.RFC3339),
			"endTime":   fleetTestTime.Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(requireRecord(t, history, fixtureReefer2042)["obdEngineStates"].([]any)) != 31 {
		t.Fatal("expected an hour of equipment engine states")
	}
}

func TestEquipmentDigitalOutput(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	target := "/fleet/equipment/" + fixtureYardSpotter + "/digital-output"
	tests := []struct {
		target   string
		body     map[string]any
		status   int
		fragment string
	}{
		{
			target:   "/fleet/equipment/abc/digital-output",
			body:     map[string]any{"pinId": float64(1), "state": true},
			status:   http.StatusBadRequest,
			fragment: "integer",
		},
		{
			target:   target,
			body:     map[string]any{"state": true},
			status:   http.StatusBadRequest,
			fragment: "pinId",
		},
		{
			target:   target,
			body:     map[string]any{"pinId": float64(1)},
			status:   http.StatusBadRequest,
			fragment: "state",
		},
		{
			target:   target,
			body:     map[string]any{"pinId": float64(0), "state": true},
			status:   http.StatusBadRequest,
			fragment: "pinId",
		},
		{
			target:   target,
			body:     map[string]any{"pinId": float64(3), "state": true},
			status:   http.StatusBadRequest,
			fragment: "1 or 2",
		},
		{
			target: target,
			body: map[string]any{
				"pinId":           float64(1),
				"state":           true,
				"durationSeconds": float64(604_801),
			},
			status:   http.StatusBadRequest,
			fragment: "durationSeconds",
		},
		{
			target:   "/fleet/equipment/" + fixtureReefer2042 + "/digital-output",
			body:     map[string]any{"pinId": float64(1), "state": true},
			status:   http.StatusBadRequest,
			fragment: "AG53",
		},
		{
			target:   "/fleet/equipment/12345/digital-output",
			body:     map[string]any{"pinId": float64(1), "state": true},
			status:   http.StatusNotFound,
			fragment: "equipment",
		},
	}
	for _, testCase := range tests {
		callAPI(
			t,
			srv,
			http.MethodPatch,
			testCase.target,
			testCase.body,
		).expectError(t, testCase.status, testCase.fragment)
	}
	data := requireStatus(t, srv, http.MethodPatch, target, map[string]any{
		"pinId": float64(2), "state": true, "durationSeconds": float64(600),
	}, http.StatusOK).data(t)
	if floatFromAny(data["id"]) != 281474980315269 || floatFromAny(data["pinId"]) != 2 ||
		data["state"] != true || floatFromAny(data["durationSeconds"]) != 600 {
		t.Fatalf("unexpected digital output response %v", data)
	}
	stored, err := srv.store.Get(ResourceAssets, fixtureYardSpotter)
	if err != nil {
		t.Fatalf("get spotter: %v", err)
	}
	if nestedAny(stored, fieldSimDigitalOutputs, "2", "state") != true {
		t.Fatalf("expected the output state to persist, got %v", stored[fieldSimDigitalOutputs])
	}
}
