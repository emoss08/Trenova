package sim

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScriptEngineLoadsAndBuildsEvents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script.yaml")
	content := `
version: 1
timezone: UTC
scenarios:
  - id: "drv-1-day"
    driverId: "drv-1"
    vehicleId: "veh-1001"
    baseDate: "2026-03-02"
    events:
      - type: "speeding.burst_major"
        start: "08:15:00"
        durationMs: 600000
        severity: "critical"
`
	if err := os.WriteFile(scriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write script fixture: %v", err)
	}

	engine := NewScriptEngine(scriptPath, "merge", "UTC")
	if err := engine.Reload(); err != nil {
		t.Fatalf("reload script: %v", err)
	}

	status := engine.Status()
	if !status.Loaded {
		t.Fatal("expected loaded status")
	}
	if status.ScenarioCount != 1 || status.EventCount != 1 {
		t.Fatalf(
			"expected 1 scenario and 1 event, got %d and %d",
			status.ScenarioCount,
			status.EventCount,
		)
	}

	start := time.Date(2026, time.March, 2, 7, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	events := engine.EventsWindow(
		start,
		end,
		map[string]struct{}{"drv-1": {}},
		map[string]struct{}{"veh-1001": {}},
	)
	if len(events) != 1 {
		t.Fatalf("expected one scripted event in range, got %d", len(events))
	}
	if events[0].Type != simEventSpeedMajor {
		t.Fatalf("unexpected event type: %q", events[0].Type)
	}
}

func TestScriptEngineRejectsDutyOverlap(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script.yaml")
	content := `
version: 1
timezone: UTC
scenarios:
  - id: "drv-1-day"
    driverId: "drv-1"
    vehicleId: "veh-1001"
    baseDate: "2026-03-02"
    events:
      - type: "duty.off_duty_pause"
        start: "08:00:00"
        durationMs: 3600000
      - type: "duty.sleeper_berth_block"
        start: "08:30:00"
        durationMs: 3600000
`
	if err := os.WriteFile(scriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write script fixture: %v", err)
	}

	engine := NewScriptEngine(scriptPath, "merge", "UTC")
	if err := engine.Reload(); err == nil {
		t.Fatal("expected duty overlap validation error")
	}
}
