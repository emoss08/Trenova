package sim

import (
	"testing"
	"time"
)

func TestLiveSimulatorAssetStreamChangesOverTime(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.startedAt.Truncate(time.Minute).Add(30 * time.Minute)
	start := now.Add(-10 * time.Minute)
	end := now

	first := simulator.AssetStream(now, []string{"veh-1"}, &start, &end)
	second := simulator.AssetStream(now.Add(4*time.Minute), []string{"veh-1"}, &start, &end)

	if len(first) == 0 {
		t.Fatal("expected at least one live asset location sample")
	}
	if len(first) != len(second) {
		t.Fatalf("expected consistent sample count, got %d and %d", len(first), len(second))
	}

	latOne := floatFromAny(nestedAny(first[0], "location", "latitude"))
	lonOne := floatFromAny(nestedAny(first[0], "location", "longitude"))
	latTwo := floatFromAny(nestedAny(second[0], "location", "latitude"))
	lonTwo := floatFromAny(nestedAny(second[0], "location", "longitude"))

	if latOne == latTwo && lonOne == lonTwo {
		t.Fatalf(
			"expected live location to move over time, got identical coordinate (%f,%f)",
			latOne,
			lonOne,
		)
	}
}

func TestLiveSimulatorHOSClocksChangeOverTime(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(3*time.Hour + 20*time.Minute)
	later := now.Add(55 * time.Minute)

	first := simulator.HOSClocks(now, []string{"drv-1"})
	second := simulator.HOSClocks(later, []string{"drv-1"})
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf(
			"expected one driver clock record in each response, got %d and %d",
			len(first),
			len(second),
		)
	}

	driveOne := floatFromAny(nestedAny(first[0], "clocks", "drive", "driveRemainingDurationMs"))
	driveTwo := floatFromAny(nestedAny(second[0], "clocks", "drive", "driveRemainingDurationMs"))
	if driveOne == driveTwo {
		t.Fatalf("expected drive remaining duration to change over time, still %f", driveOne)
	}
}

func TestLiveSimulatorHOSClocksRespectELDLimits(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(36*time.Hour + 12*time.Minute)

	records := simulator.HOSClocks(now, []string{"drv-1"})
	if len(records) != 1 {
		t.Fatalf("expected one clock record, got %d", len(records))
	}

	record := records[0]
	driveRemaining := floatFromAny(nestedAny(record, "clocks", "drive", "driveRemainingDurationMs"))
	shiftRemaining := floatFromAny(nestedAny(record, "clocks", "shift", "shiftRemainingDurationMs"))
	breakRemaining := floatFromAny(nestedAny(record, "clocks", "break", "timeUntilBreakDurationMs"))
	cycleRemaining := floatFromAny(nestedAny(record, "clocks", "cycle", "cycleRemainingDurationMs"))

	if driveRemaining < 0 || driveRemaining > float64(eldDriveLimit.Milliseconds()) {
		t.Fatalf("drive remaining out of ELD bounds: %f", driveRemaining)
	}
	if shiftRemaining < 0 || shiftRemaining > float64(eldShiftLimit.Milliseconds()) {
		t.Fatalf("shift remaining out of ELD bounds: %f", shiftRemaining)
	}
	if breakRemaining < 0 || breakRemaining > float64(eldBreakLimit.Milliseconds()) {
		t.Fatalf("break remaining out of ELD bounds: %f", breakRemaining)
	}
	if cycleRemaining < 0 || cycleRemaining > float64(eldCycleLimit.Milliseconds()) {
		t.Fatalf("cycle remaining out of ELD bounds: %f", cycleRemaining)
	}
}

func TestLiveSimulatorHOSClocksAreDrivingWhenAssignedVehicleMoves(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	movingSamples := 0
	for hour := 0; hour < 24; hour += 2 {
		now := simulator.anchorTime.Add(time.Duration(hour) * time.Hour)
		if !simulator.shouldForceDriving("veh-1", now, now) {
			continue
		}
		movingSamples++

		records := simulator.HOSClocks(now, []string{"drv-1"})
		if len(records) != 1 {
			t.Fatalf("expected one clock record, got %d", len(records))
		}
		status := nestedString(records[0], "currentDutyStatus", "hosStatusType")
		if status != hosStatusDriving {
			t.Fatalf(
				"expected driving status while vehicle is moving at %s, got %q",
				now.UTC().Format(time.RFC3339),
				status,
			)
		}
	}
	if movingSamples == 0 {
		t.Fatal("expected at least one moving sample for veh-1")
	}
}

func TestLiveSimulatorUnassignedDriverIsNotDriving(t *testing.T) {
	t.Parallel()

	fixture := &Fixture{
		Drivers: []Record{
			{
				"id":   "drv-unassigned",
				"name": "Casey Nguyen",
			},
		},
		HOSClocks: []Record{
			{
				"driver": map[string]any{
					"id":   "drv-unassigned",
					"name": "Casey Nguyen",
				},
			},
		},
	}
	simulator := NewLiveSimulator(NewStore(fixture), "unassigned-driver")

	now := simulator.anchorTime.Add(37 * time.Hour)
	records := simulator.HOSClocks(now, []string{"drv-unassigned"})
	if len(records) != 1 {
		t.Fatalf("expected one clock record, got %d", len(records))
	}
	status := nestedString(records[0], "currentDutyStatus", "hosStatusType")
	if status == hosStatusDriving {
		t.Fatal("unassigned driver must not be in driving status")
	}
}

func TestLiveSimulatorAutoAssignsUnassignedDriverVehicle(t *testing.T) {
	t.Parallel()

	fixture := &Fixture{
		Assets: []Record{
			{
				"id":   "veh-fallback",
				"name": "Fallback Truck",
				"type": "vehicle",
			},
		},
		Drivers: []Record{
			{
				"id":   "drv-fallback",
				"name": "Fallback Driver",
			},
		},
		HOSClocks: []Record{
			{
				"driver": map[string]any{
					"id":   "drv-fallback",
					"name": "Fallback Driver",
				},
			},
		},
	}

	simulator := NewLiveSimulator(NewStore(fixture), "fallback-seed")
	records := simulator.HOSClocks(simulator.anchorTime.Add(time.Hour), []string{"drv-fallback"})
	if len(records) != 1 {
		t.Fatalf("expected one HOS clock record, got %d", len(records))
	}

	vehicleID := nestedString(records[0], "currentVehicle", "id")
	if vehicleID != "veh-fallback" {
		t.Fatalf("expected auto-assigned vehicle veh-fallback, got %q", vehicleID)
	}
}

func TestLiveSimulatorHOSLogsAdvanceWithTime(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(8 * time.Hour)
	later := now.Add(5 * time.Hour)

	first := simulator.HOSLogs(now, []string{"drv-1"}, nil, nil)
	second := simulator.HOSLogs(later, []string{"drv-1"}, nil, nil)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf(
			"expected one driver log record in each response, got %d and %d",
			len(first),
			len(second),
		)
	}

	latestOne := mustLatestHOSLogStart(t, first[0])
	latestTwo := mustLatestHOSLogStart(t, second[0])
	if !latestTwo.After(latestOne) {
		t.Fatalf(
			"expected later response to include newer log starts, got %s then %s",
			latestOne,
			latestTwo,
		)
	}
}

func TestLiveSimulatorHOSLogsDrivingWhenVehicleMoving(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(26 * time.Hour)
	start := now.Add(-8 * time.Hour)
	end := now

	records := simulator.HOSLogs(now, []string{"drv-1"}, &start, &end)
	if len(records) != 1 {
		t.Fatalf("expected one HOS log record, got %d", len(records))
	}

	rawLogs, ok := records[0]["hosLogs"].([]any)
	if !ok {
		t.Fatal("expected hosLogs array")
	}

	for _, raw := range rawLogs {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			continue
		}
		startTimeValue := stringValue(entry, "logStartTime")
		if startTimeValue == "" {
			continue
		}
		logStart, parseErr := time.Parse(time.RFC3339, startTimeValue)
		if parseErr != nil {
			t.Fatalf("failed parsing logStartTime %q: %v", startTimeValue, parseErr)
		}

		if !simulator.shouldForceDriving("veh-1", logStart, now) {
			continue
		}

		status := stringValue(entry, "hosStatusType")
		if status != hosStatusDriving {
			t.Fatalf(
				"expected driving status for moving vehicle at %s, got %q",
				logStart.UTC().Format(time.RFC3339),
				status,
			)
		}
	}
}

func TestLiveSimulatorHOSLogsUnassignedDriverNeverDriving(t *testing.T) {
	t.Parallel()

	fixture := &Fixture{
		Drivers: []Record{
			{
				"id":   "drv-unassigned",
				"name": "Casey Nguyen",
			},
		},
		HOSLogs: []Record{
			{
				"driver": map[string]any{
					"id":   "drv-unassigned",
					"name": "Casey Nguyen",
				},
			},
		},
	}
	simulator := NewLiveSimulator(NewStore(fixture), "unassigned-driver-logs")

	now := simulator.anchorTime.Add(44 * time.Hour)
	start := now.Add(-24 * time.Hour)
	end := now
	records := simulator.HOSLogs(now, []string{"drv-unassigned"}, &start, &end)
	if len(records) != 1 {
		t.Fatalf("expected one HOS log record, got %d", len(records))
	}
	rawLogs, ok := records[0]["hosLogs"].([]any)
	if !ok {
		t.Fatal("expected hosLogs array")
	}
	for _, raw := range rawLogs {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			continue
		}
		if stringValue(entry, "hosStatusType") == hosStatusDriving {
			t.Fatal("unassigned driver must not have driving hos logs")
		}
	}
}

func TestLiveSimulatorAssetMovementBoundedPerSecond(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.startedAt.Add(2 * time.Hour)
	sampleTime := now.Add(-8 * time.Minute)

	first := simulator.routeStateForSingleTime("veh-1", sampleTime, now)
	second := simulator.routeStateForSingleTime("veh-1", sampleTime, now.Add(time.Second))

	distanceMeters := haversineMeters(
		first.Latitude,
		first.Longitude,
		second.Latitude,
		second.Longitude,
	)

	allowed := (maxRouteSpeedMPS * 1.3) + (2 * defaultGPSJitterMeters)
	if distanceMeters > allowed {
		t.Fatalf(
			"expected <= %.2fm movement in one second, got %.2fm",
			allowed,
			distanceMeters,
		)
	}
}

func TestBuildRouteSegmentsDoesNotShortcutEndToStart(t *testing.T) {
	t.Parallel()

	points := []routePoint{
		{Latitude: 30.0000, Longitude: -97.0000, SpeedMPS: 20},
		{Latitude: 30.0100, Longitude: -97.0100, SpeedMPS: 20},
		{Latitude: 30.0200, Longitude: -97.0200, SpeedMPS: 20},
		{Latitude: 30.0300, Longitude: -97.0300, SpeedMPS: 20},
	}

	segments, period := buildRouteSegments(points)
	if period <= 0 {
		t.Fatal("expected positive period")
	}
	if len(segments) != 6 {
		t.Fatalf("expected 6 segments for out-and-back route, got %d", len(segments))
	}

	for _, segment := range segments {
		if segment.From.Latitude == points[len(points)-1].Latitude &&
			segment.From.Longitude == points[len(points)-1].Longitude &&
			segment.To.Latitude == points[0].Latitude &&
			segment.To.Longitude == points[0].Longitude {
			t.Fatal("unexpected direct end-to-start shortcut segment")
		}
	}
}

func TestTuneRouteSegmentsForDurationAppliesTripHourFloor(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	simulator.options.TripHoursMin = 8
	simulator.options.TripHoursMax = 12

	points := simulator.loadAssetWaypoints()["veh-1"]
	baseSegments, basePeriod := buildRouteSegments(points)
	if len(baseSegments) == 0 || basePeriod <= 0 {
		t.Fatal("expected baseline route segments")
	}

	scaledSegments, scaledPeriod := simulator.tuneRouteSegmentsForDuration(
		"veh-1",
		baseSegments,
		basePeriod,
	)
	if len(scaledSegments) != len(baseSegments) {
		t.Fatalf(
			"expected same segment count after scaling, got %d and %d",
			len(scaledSegments),
			len(baseSegments),
		)
	}
	if scaledPeriod < 8*time.Hour {
		t.Fatalf("expected scaled period >= 8h, got %s", scaledPeriod)
	}
	if scaledPeriod < basePeriod {
		t.Fatalf(
			"expected scaled period >= base period, got base=%s scaled=%s",
			basePeriod,
			scaledPeriod,
		)
	}
}

func TestRouteLoopTargetPeriodFallsWithinConfiguredRange(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	simulator.options.TripHoursMin = 8
	simulator.options.TripHoursMax = 12

	target := simulator.routeLoopTargetPeriod("veh-1")
	if target < 8*time.Hour || target > 12*time.Hour {
		t.Fatalf("expected target period in [8h,12h], got %s", target)
	}
}

func TestTuneRouteSegmentsForDurationLeavesLongRoutesUnchanged(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	simulator.options.TripHoursMin = 8
	simulator.options.TripHoursMax = 12

	segments := []routeSegment{
		{
			DistanceMeters:  30_000,
			SpeedMPS:        10,
			Duration:        4 * time.Hour,
			CumulativeStart: 0,
		},
		{
			DistanceMeters:  30_000,
			SpeedMPS:        10,
			Duration:        5 * time.Hour,
			CumulativeStart: 4 * time.Hour,
		},
	}
	basePeriod := 14 * time.Hour

	scaledSegments, scaledPeriod := simulator.tuneRouteSegmentsForDuration(
		"veh-long",
		segments,
		basePeriod,
	)
	if scaledPeriod != basePeriod {
		t.Fatalf("expected unchanged long period %s, got %s", basePeriod, scaledPeriod)
	}
	if scaledSegments[0].Duration != segments[0].Duration ||
		scaledSegments[1].Duration != segments[1].Duration {
		t.Fatal("expected unchanged long-route segment durations")
	}
}

func TestLiveSimulatorRoutesIncludeLifecycleAndStops(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(4 * time.Hour)
	routes := simulator.Routes(now, []string{"route-1"}, nil)
	if len(routes) != 1 {
		t.Fatalf("expected one route, got %d", len(routes))
	}

	route := routes[0]
	status := stringValue(route, "status")
	if status == "" {
		t.Fatal("expected dynamic route status to be populated")
	}
	lifecycle, ok := anyAsMap(route["lifecycle"])
	if !ok {
		t.Fatal("expected lifecycle object in route payload")
	}
	if stringValue(Record(lifecycle), "status") == "" {
		t.Fatal("expected lifecycle.status to be populated")
	}
	rawStops, ok := route["stops"].([]any)
	if !ok {
		t.Fatal("expected stops array in route payload")
	}
	if len(rawStops) < 3 {
		t.Fatalf("expected at least 3 stops, got %d", len(rawStops))
	}
	firstStop, ok := rawStops[0].(map[string]any)
	if !ok {
		t.Fatal("expected stop payload map")
	}
	if stringValue(Record(firstStop), "etaTime") == "" {
		t.Fatal("expected stop etaTime to be populated")
	}
	if stringValue(Record(firstStop), "scheduledWindowStartTime") == "" {
		t.Fatal("expected stop scheduledWindowStartTime to be populated")
	}
}

func TestLiveSimulatorRoutesFilterByStatus(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(5 * time.Hour)
	routes := simulator.Routes(now, []string{"route-1"}, nil)
	if len(routes) != 1 {
		t.Fatalf("expected one route, got %d", len(routes))
	}
	status := stringValue(routes[0], "status")
	if status == "" {
		t.Fatal("expected route status")
	}

	filtered := simulator.Routes(now, []string{"route-1"}, []string{status})
	if len(filtered) != 1 {
		t.Fatalf("expected route to match status filter %q, got %d records", status, len(filtered))
	}
}

func mustLatestHOSLogStart(t *testing.T, record Record) time.Time {
	t.Helper()

	rawLogs, ok := record["hosLogs"].([]any)
	if !ok {
		t.Fatal("expected hosLogs to be a slice")
	}
	latest := time.Time{}
	for _, raw := range rawLogs {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			continue
		}
		start := stringValue(entry, "logStartTime")
		if start == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, start)
		if err != nil {
			continue
		}
		if parsed.After(latest) {
			latest = parsed
		}
	}
	if latest.IsZero() {
		t.Fatal("expected at least one parsable hos log start time")
	}
	return latest.UTC()
}

func newTestLiveSimulator() *LiveSimulator {
	fixture := &Fixture{
		Assets: []Record{
			{
				"id":   "veh-1",
				"name": "Truck 1001",
				"type": "vehicle",
				"externalIds": map[string]any{
					"tmsVehicleId": "unit-1001",
				},
			},
		},
		AssetLocation: []Record{
			{
				"asset": map[string]any{
					"id": "veh-1",
				},
				"happenedAtTime": "2026-03-01T14:00:00Z",
				"location": map[string]any{
					"latitude":       30.2672,
					"longitude":      -97.7431,
					"headingDegrees": 88.0,
					"address": map[string]any{
						"city":    "Austin",
						"state":   "TX",
						"country": "USA",
					},
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 18.2,
					"ecuSpeedMetersPerSecond": 17.9,
				},
			},
			{
				"asset": map[string]any{
					"id": "veh-1",
				},
				"happenedAtTime": "2026-03-01T14:08:00Z",
				"location": map[string]any{
					"latitude":       30.3001,
					"longitude":      -97.7004,
					"headingDegrees": 92.0,
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 22.3,
					"ecuSpeedMetersPerSecond": 22.1,
				},
			},
		},
		Drivers: []Record{
			{
				"id":   "drv-1",
				"name": "Alex Rivera",
			},
		},
		Routes: []Record{
			{
				"id":   "route-1",
				"name": "Austin Loop",
				"driver": map[string]any{
					"id":   "drv-1",
					"name": "Alex Rivera",
				},
				"vehicle": map[string]any{
					"id":   "veh-1",
					"name": "Truck 1001",
				},
			},
		},
		VehicleStats: []Record{
			{
				"id":   "veh-1",
				"name": "Truck 1001",
				"fuelPercent": map[string]any{
					"value": 87.0,
				},
				"obdEngineSeconds": map[string]any{
					"value": 5_412_000.0,
				},
				"batteryMilliVolts": map[string]any{
					"value": 12_870.0,
				},
			},
		},
		HOSClocks: []Record{
			{
				"driver": map[string]any{
					"id":   "drv-1",
					"name": "Alex Rivera",
				},
				"currentVehicle": map[string]any{
					"id":   "veh-1",
					"name": "Truck 1001",
				},
			},
		},
	}
	store := NewStore(fixture)
	return NewLiveSimulator(store, "test-live-seed")
}
