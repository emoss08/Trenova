package sim

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

func TestVehicleStatsTypeValidation(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	for _, endpoint := range []string{
		"/fleet/vehicles/stats",
		"/fleet/vehicles/stats/feed",
		"/fleet/vehicles/stats/history?startTime=2026-03-04T14:00:00Z&endTime=2026-03-04T15:00:00Z",
	} {
		separator := "?"
		if slices.Contains([]byte(endpoint), '?') {
			separator = "&"
		}
		callAPI(
			t,
			srv,
			http.MethodGet,
			endpoint,
			nil,
		).expectError(t, http.StatusBadRequest, "types")
		callAPI(t, srv, http.MethodGet, endpoint+separator+"types=gps,warpDrive", nil).
			expectError(t, http.StatusBadRequest, "warpDrive")
		callAPI(
			t,
			srv,
			http.MethodGet,
			endpoint+separator+"types=gps,engineStates,fuelPercents,obdOdometerMeters",
			nil,
		).
			expectError(t, http.StatusBadRequest, "up to 3")
		callAPI(
			t,
			srv,
			http.MethodGet,
			endpoint+separator+"types=auxInput1,auxInput2,gps,auxInput3",
			nil,
		).
			expectError(t, http.StatusBadRequest, "up to 3")
		requireStatus(
			t,
			srv,
			http.MethodGet,
			endpoint+separator+"types=engineStates,obdOdometerMeters,auxInput3,auxInput4,auxInput10",
			nil,
			http.StatusOK,
		)
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&decorations=gps,fuelPercents,engineStates",
		nil,
	).
		expectError(t, http.StatusBadRequest, "up to 2")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&decorations=gpsOdometerMeters",
		nil,
	).
		expectError(t, http.StatusBadRequest, "decoration")
	callAPI(t, srv, http.MethodGet, "/fleet/vehicles/stats?types=gps&time=noon", nil).
		expectError(t, http.StatusBadRequest, "time")
}

func TestVehicleStatsSnapshotReturnsOnlyRequestedTypes(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	rows := requireStatus(t, srv, http.MethodGet,
		"/fleet/vehicles/stats?types=gps,engineStates,fuelPercents", nil, http.StatusOK).list(t)
	if len(rows) != 12 {
		t.Fatalf("expected a row per tractor, got %d", len(rows))
	}
	allowed := []string{"id", "name", fieldExternalIDs, "gps", "engineState", "fuelPercent"}
	for _, row := range rows {
		for key := range row {
			if !slices.Contains(allowed, key) {
				t.Fatalf("expected only requested stats, found %q on %s", key, recordID(row))
			}
		}
		gps, ok := anyAsMap(row["gps"])
		if !ok {
			t.Fatalf("expected gps on %s", recordID(row))
		}
		if nestedString(Record(gps), "reverseGeo", "formattedLocation") == "" ||
			stringValue(Record(gps), "time") != fleetTestTime.Format(time.RFC3339) {
			t.Fatalf("expected reverse-geocoded gps at the sim time, got %v", gps)
		}
		if nestedString(row, fieldExternalIDs, externalIDVinKey) == "" {
			t.Fatalf("expected externalIds on stats rows, got %v", row[fieldExternalIDs])
		}
		engine := nestedString(row, "engineState", "value")
		if !slices.Contains([]string{engineStateOn, engineStateOff, engineStateIdle}, engine) {
			t.Fatalf("unexpected engine state %q", engine)
		}
		if (engine == engineStateOn) != (floatFromAny(gps["speedMilesPerHour"]) > movingSpeedThresholdMPS*2.23694) {
			t.Fatalf(
				"expected engine state to agree with speed: %s at %v mph",
				engine,
				gps["speedMilesPerHour"],
			)
		}
	}

	unsupported := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=evStateOfChargeMilliPercent,spreaderActive,tellTales",
		nil,
		http.StatusOK,
	).list(t)
	for _, row := range unsupported {
		if len(row) != 3 {
			t.Fatalf("expected diesel tractors to report no EV or spreader data, got %v", row)
		}
	}

	past := fleetTestTime.Add(-3 * time.Hour)
	historic := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=obdOdometerMeters&vehicleIds="+fixtureTruck1001+"&time="+past.Format(
			time.RFC3339,
		),
		nil,
		http.StatusOK,
	).list(t)
	current := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=obdOdometerMeters&vehicleIds="+fixtureTruck1001,
		nil,
		http.StatusOK,
	).list(t)
	if floatFromAny(nestedAny(historic[0], "obdOdometerMeters", "value")) >=
		floatFromAny(nestedAny(current[0], "obdOdometerMeters", "value")) {
		t.Fatal("expected the odometer to grow between the time parameter and now")
	}
	if nestedString(historic[0], "obdOdometerMeters", "time") != past.Format(time.RFC3339) {
		t.Fatalf(
			"expected the sample at the requested time, got %v",
			historic[0]["obdOdometerMeters"],
		)
	}
	future := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?types=gps&vehicleIds="+fixtureTruck1001+"&time=2030-01-01T00:00:00Z",
		nil,
		http.StatusOK,
	).list(t)
	if nestedString(future[0], "gps", "time") != fleetTestTime.Format(time.RFC3339) {
		t.Fatal("expected a future time to return the latest data")
	}

	every := []string{}
	for _, statType := range vehicleStatRegistry.order {
		row := requireStatus(t, srv, http.MethodGet,
			"/fleet/vehicles/stats?types="+statType+"&vehicleIds="+fixtureTruck1001, nil, http.StatusOK).list(t)[0]
		if _, has := row[vehicleStatRegistry.specs[statType].SnapshotKey]; has {
			every = append(every, statType)
		}
	}
	for _, want := range []string{
		"ambientAirTemperatureMilliC", "barometricPressurePa", "batteryMilliVolts", "defLevelMilliPercent",
		"ecuDoorStatus", "ecuSpeedMph", "engineCoolantTemperatureMilliC", "engineImmobilizer",
		"engineLoadPercent", "engineOilPressureKPa", "engineRpm", "engineStates", "faultCodes",
		"fuelPercents", "fuelConsumedMilliliters", "gps", "gpsDistanceMeters",
		"idlingDurationMilliseconds", "intakeManifoldTemperatureMilliC", "nfcCardScans",
		"obdEngineSeconds", "obdOdometerMeters", "seatbeltDriver",
	} {
		if !slices.Contains(every, want) {
			t.Fatalf("expected Truck 1001 to report %s, got %v", want, every)
		}
	}
	if len(vehicleStatRegistry.order) != 63 {
		t.Fatalf("expected the 63 spec stat types, got %d", len(vehicleStatRegistry.order))
	}
}

func TestVehicleStatsFeedDecorationsAndEvents(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	first := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=engineStates,nfcCardScans&decorations=gps,fuelPercents&vehicleIds="+fixtureTruck1001,
		nil,
		http.StatusOK,
	)
	row := first.list(t)[0]
	samples := row["engineStates"].([]any)
	if len(samples) != 1 {
		t.Fatalf("expected the latest engine state on the first call, got %d", len(samples))
	}
	decorations, ok := anyAsMap(samples[0].(map[string]any)[fieldDecorations])
	if !ok {
		t.Fatalf("expected decorations on feed samples, got %v", samples[0])
	}
	gps, hasGPS := anyAsMap(decorations["gps"])
	fuel, hasFuel := anyAsMap(decorations["fuelPercents"])
	if !hasGPS || !hasFuel || gps["time"] != nil || fuel["value"] == nil {
		t.Fatalf("expected gps and fuel decorations without their own time, got %v", decorations)
	}
	scans := row["nfcCardScans"].([]any)
	if len(scans) != 1 ||
		nestedString(Record(scans[0].(map[string]any)), "card", "id") != "941767043" {
		t.Fatalf("expected Alex's last card scan, got %v", scans)
	}

	snapshot := requireStatus(t, srv, http.MethodGet,
		"/fleet/vehicles/stats?types=gps&vehicleIds="+fixtureTruck1001, nil, http.StatusOK).list(t)[0]
	feedGPS := requireStatus(t, srv, http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&vehicleIds="+fixtureTruck1001, nil, http.StatusOK).list(t)[0]["gps"].([]any)[0]
	if floatFromAny(
		nestedAny(snapshot, "gps", "latitude"),
	) != floatFromAny(
		feedGPS.(map[string]any)["latitude"],
	) {
		t.Fatalf(
			"expected snapshot and feed to agree at the same instant, got %v vs %v",
			snapshot["gps"],
			feedGPS,
		)
	}

	cursor := stringValue(first.pagination(t), "endCursor")
	srv.clock.Step(26 * time.Hour)
	next := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=nfcCardScans,faultCodes&vehicleIds="+fixtureTruck1001+"&after="+cursor,
		nil,
		http.StatusOK,
	)
	nextRow := next.list(t)[0]
	cursorTime, _, err := decodeStatsFeedCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	for _, statType := range []string{"nfcCardScans", "faultCodes"} {
		for _, raw := range nextRow[statType].([]any) {
			at := mustRFC3339(t, stringValue(Record(raw.(map[string]any)), "time"))
			if !at.After(cursorTime) {
				t.Fatalf("expected %s events after the cursor, got %s", statType, at)
			}
		}
	}
	if next.pagination(t)["hasNextPage"] != true {
		t.Fatal("expected a long gap to page through the feed")
	}
}

func TestVehicleStatsHistoryCadence(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	rows := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/vehicles/stats/history", map[string]string{
			"types":       "gps,obdOdometerMeters",
			"decorations": "engineStates",
			"vehicleIds":  fixtureTruck1001,
			"startTime":   fleetTestTime.Add(-30 * time.Minute).Format(time.RFC3339),
			"endTime":     fleetTestTime.Add(30 * time.Minute).Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).list(t)
	samples := rows[0]["obdOdometerMeters"].([]any)
	if len(samples) != 16 {
		t.Fatalf("expected 2-minute samples up to now (future clipped), got %d", len(samples))
	}
	previous := 0.0
	for _, raw := range samples {
		sample := Record(raw.(map[string]any))
		value := floatFromAny(sample["value"])
		if value < previous {
			t.Fatal("expected a monotonic odometer")
		}
		previous = value
		if nestedString(sample, fieldDecorations, "engineStates", "value") == "" {
			t.Fatalf("expected engine state decorations, got %v", sample)
		}
	}
}
