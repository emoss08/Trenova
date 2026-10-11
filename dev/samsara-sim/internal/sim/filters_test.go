package sim

import (
	"testing"
	"time"
)

func TestFilterAssetLocationRecords(t *testing.T) {
	t.Parallel()

	records := []Record{
		{
			"asset":          map[string]any{"id": testVehicleID},
			"happenedAtTime": "2026-03-01T14:00:00Z",
		},
		{
			"asset":          map[string]any{"id": "281474976710658"},
			"happenedAtTime": "2026-03-01T14:05:00Z",
		},
		{
			"asset":          map[string]any{"id": testVehicleID},
			"happenedAtTime": "2026-03-01T14:10:00Z",
		},
	}

	start := mustParseRFC3339(t, "2026-03-01T14:02:00Z")
	end := mustParseRFC3339(t, "2026-03-01T14:15:00Z")

	filtered := filterAssetLocationRecords(records, []string{testVehicleID}, &start, &end)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered record, got %d", len(filtered))
	}
	if nestedString(filtered[0], "asset", "id") != testVehicleID {
		t.Fatalf("unexpected asset id: %s", nestedString(filtered[0], "asset", "id"))
	}
	if stringValue(filtered[0], "happenedAtTime") != "2026-03-01T14:10:00Z" {
		t.Fatalf("unexpected happenedAtTime: %s", stringValue(filtered[0], "happenedAtTime"))
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
