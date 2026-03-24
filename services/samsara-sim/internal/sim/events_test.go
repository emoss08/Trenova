package sim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLiveSimulatorEventsWindowDeterministic(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	start := simulator.anchorTime.Add(-12 * time.Hour)
	end := simulator.anchorTime.Add(24 * time.Hour)

	first := simulator.EventsWindow(start, end, []string{"drv-1"}, []string{"veh-1"}, 0)
	second := simulator.EventsWindow(start, end, []string{"drv-1"}, []string{"veh-1"}, 0)
	if len(first) == 0 {
		t.Fatal("expected deterministic event set to contain events")
	}
	if len(first) != len(second) {
		t.Fatalf("expected identical event counts, got %d and %d", len(first), len(second))
	}
	for idx := range first {
		if first[idx].ID != second[idx].ID {
			t.Fatalf("event mismatch at index %d: %q != %q", idx, first[idx].ID, second[idx].ID)
		}
	}
}

func TestApplyVehicleEventsToRouteStateStopFreezesPosition(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	waypoints := simulator.loadAssetWaypoints()
	points := waypoints["veh-1"]
	if len(points) < 2 {
		t.Fatal("expected test vehicle waypoints")
	}

	windowStart := simulator.anchorTime.Add(-30 * time.Minute)
	now := simulator.anchorTime.Add(45 * time.Minute)
	sampleTime := now
	stopStart := now.Add(-3 * time.Minute)

	baseAtSample := simulator.routeStateForSample("veh-1", points, sampleTime, windowStart, now)
	baseAtStop := simulator.routeStateForSample("veh-1", points, stopStart, windowStart, now)
	withEvent := simulator.applyVehicleEventsToRouteState(
		"veh-1",
		points,
		[]SimEvent{
			{
				ID:        "evt-stop",
				Type:      simEventStopFuelBreak,
				DriverID:  "drv-1",
				VehicleID: "veh-1",
				StartsAt:  stopStart,
				EndsAt:    stopStart.Add(20 * time.Minute),
				Severity:  "info",
			},
		},
		sampleTime,
		windowStart,
		now,
		baseAtSample,
	)

	if withEvent.SpeedMPS != 0 {
		t.Fatalf("expected stopped speed during stop event, got %.2f", withEvent.SpeedMPS)
	}
	if withEvent.Latitude != baseAtStop.Latitude || withEvent.Longitude != baseAtStop.Longitude {
		t.Fatalf(
			"expected frozen location at stop start (%.6f,%.6f), got (%.6f,%.6f)",
			baseAtStop.Latitude,
			baseAtStop.Longitude,
			withEvent.Latitude,
			withEvent.Longitude,
		)
	}
}

func TestDutyStatusForSimEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		eventType string
		fallback  string
		want      string
	}{
		{
			name:      "off duty event",
			eventType: simEventDutyOffDutyPause,
			fallback:  hosStatusDriving,
			want:      hosStatusOffDuty,
		},
		{
			name:      "sleeper event",
			eventType: simEventDutySleeperBlock,
			fallback:  hosStatusDriving,
			want:      hosStatusSleeperBed,
		},
		{
			name:      "speed event",
			eventType: simEventSpeedMinor,
			fallback:  hosStatusOnDuty,
			want:      hosStatusDriving,
		},
		{
			name:      "violation keeps fallback",
			eventType: simEventViolationDrive,
			fallback:  hosStatusOnDuty,
			want:      hosStatusOnDuty,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := dutyStatusForSimEvent(
				&SimEvent{Type: tc.eventType},
				tc.fallback,
			)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestActiveEventSummary(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	window := simulator.EventsWindow(
		simulator.anchorTime.Add(-24*time.Hour),
		simulator.anchorTime.Add(48*time.Hour),
		nil,
		nil,
		0,
	)
	if len(window) == 0 {
		t.Fatal("expected at least one generated event")
	}
	at := window[0].StartsAt.Add(time.Minute)
	byType, violations, speeding := simulator.ActiveEventSummary(at)
	if len(byType) == 0 {
		t.Fatal("expected active event summary to include at least one event type")
	}
	if violations < 0 {
		t.Fatalf("expected non-negative violations count, got %d", violations)
	}
	if speeding < 0 {
		t.Fatalf("expected non-negative speeding count, got %d", speeding)
	}
}

func TestEventsWindowScriptOverride(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	simulator.options.ScriptMode = scriptModeOverride
	dayStart := simulator.anchorTime.UTC().Truncate(24 * time.Hour)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "override.yaml")
	content := "version: 1\n" +
		"timezone: UTC\n" +
		"scenarios:\n" +
		"  - id: drv-1-override\n" +
		"    driverId: drv-1\n" +
		"    vehicleId: veh-1\n" +
		"    baseDate: \"" + dayStart.Format("2006-01-02") + "\"\n" +
		"    events:\n" +
		"      - type: speeding.burst_minor\n" +
		"        start: \"09:00:00\"\n" +
		"        durationMs: 600000\n" +
		"        severity: warning\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write override script: %v", err)
	}

	engine := NewScriptEngine(scriptPath, "override", "UTC")
	if err := engine.Reload(); err != nil {
		t.Fatalf("reload script: %v", err)
	}
	simulator.SetScriptEngine(engine)

	events := simulator.EventsWindow(
		dayStart,
		dayStart.Add(24*time.Hour),
		[]string{"drv-1"},
		[]string{"veh-1"},
		0,
	)
	foundScript := false
	for _, event := range events {
		if event.StartsAt.UTC().Truncate(24*time.Hour).Equal(dayStart) &&
			!strings.HasPrefix(event.ID, "evt-script-") {
			t.Fatalf("expected overridden day event to be scripted, got %q", event.ID)
		}
		if strings.HasPrefix(event.ID, "evt-script-") && event.Type == simEventSpeedMinor {
			foundScript = true
		}
	}
	if !foundScript {
		t.Fatal("expected scripted override event in response")
	}
}
