package sim

import (
	"testing"
	"time"
)

func TestFilterAssetLocationRecords(t *testing.T) {
	t.Parallel()

	records := []Record{
		{
			"asset":          map[string]any{"id": "veh-1"},
			"happenedAtTime": "2026-03-01T14:00:00Z",
		},
		{
			"asset":          map[string]any{"id": "veh-2"},
			"happenedAtTime": "2026-03-01T14:05:00Z",
		},
		{
			"asset":          map[string]any{"id": "veh-1"},
			"happenedAtTime": "2026-03-01T14:10:00Z",
		},
	}

	start := mustParseRFC3339(t, "2026-03-01T14:02:00Z")
	end := mustParseRFC3339(t, "2026-03-01T14:15:00Z")

	filtered := filterAssetLocationRecords(records, []string{"veh-1"}, &start, &end)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered record, got %d", len(filtered))
	}
	if nestedString(filtered[0], "asset", "id") != "veh-1" {
		t.Fatalf("unexpected asset id: %s", nestedString(filtered[0], "asset", "id"))
	}
	if stringValue(filtered[0], "happenedAtTime") != "2026-03-01T14:10:00Z" {
		t.Fatalf("unexpected happenedAtTime: %s", stringValue(filtered[0], "happenedAtTime"))
	}
}

func TestFilterHOSLogsByTimeWindow(t *testing.T) {
	t.Parallel()

	records := []Record{
		{
			"driver": map[string]any{"id": "drv-1"},
			"hosLogs": []any{
				map[string]any{"logStartTime": "2026-03-01T08:00:00Z", "hosStatusType": "offDuty"},
				map[string]any{"logStartTime": "2026-03-01T10:00:00Z", "hosStatusType": "driving"},
				map[string]any{"logStartTime": "2026-03-01T14:00:00Z", "hosStatusType": "onDuty"},
			},
		},
	}

	start := mustParseRFC3339(t, "2026-03-01T09:00:00Z")
	end := mustParseRFC3339(t, "2026-03-01T12:00:00Z")
	filtered := filterHOSLogsByTimeWindow(records, &start, &end)
	if len(filtered) != 1 {
		t.Fatalf("expected one driver record, got %d", len(filtered))
	}

	logs, ok := filtered[0]["hosLogs"].([]any)
	if !ok {
		t.Fatal("expected hosLogs slice in filtered output")
	}
	if len(logs) != 1 {
		t.Fatalf("expected one filtered hos log, got %d", len(logs))
	}

	entry, ok := logs[0].(map[string]any)
	if !ok {
		t.Fatal("expected hos log entry map")
	}
	if stringValue(entry, "logStartTime") != "2026-03-01T10:00:00Z" {
		t.Fatalf("unexpected logStartTime: %s", stringValue(entry, "logStartTime"))
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %s: %v", value, err)
	}
	return parsed.UTC()
}
