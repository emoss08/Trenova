package sim

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func newGeofenceFixture() *Fixture {
	return &Fixture{
		Addresses: []Record{
			{
				"id":               "addr-geo",
				"name":             "Austin Geofenced Yard",
				"formattedAddress": "1 Yard Rd, Austin, TX 78701",
				"externalIds": map[string]any{
					"tmsLocationId": "loc-geo",
				},
				"geofence": map[string]any{
					"circle": map[string]any{
						"latitude":     30.0,
						"longitude":    -97.0,
						"radiusMeters": 4000.0,
					},
				},
			},
		},
		Assets: []Record{
			{
				"id":           "veh-geo",
				"name":         "Truck Geo",
				"type":         "vehicle",
				"licensePlate": "TX-GEO1",
				"vin":          "1FUJGLDR5CLBP9999",
				"externalIds": map[string]any{
					"tmsVehicleId": "unit-geo",
				},
			},
		},
		AssetLocation: []Record{
			{
				"asset": map[string]any{"id": "veh-geo"},
				"location": map[string]any{
					"latitude":       30.0,
					"longitude":      -97.0,
					"headingDegrees": 90.0,
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 12.5,
				},
				"happenedAtTime": "2026-03-01T14:00:00Z",
			},
			{
				"asset": map[string]any{"id": "veh-geo"},
				"location": map[string]any{
					"latitude":       30.0,
					"longitude":      -96.8,
					"headingDegrees": 90.0,
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 12.5,
				},
				"happenedAtTime": "2026-03-01T14:20:00Z",
			},
		},
	}
}

func newGeofenceLiveSimulator() *LiveSimulator {
	return NewLiveSimulator(NewStore(newGeofenceFixture()), "geo-seed", LiveSimulationOptions{
		FleetSize:    1,
		TripHoursMin: 1,
		TripHoursMax: 1,
	})
}

func TestGeofenceWebhookEmissionsDetectsEntryAndExit(t *testing.T) {
	t.Parallel()

	live := newGeofenceLiveSimulator()
	now := time.Now().UTC()
	emissions := live.GeofenceWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	if len(emissions) == 0 {
		t.Fatal("expected geofence transitions across a full route loop")
	}

	sawEntry := false
	sawExit := false
	for idx := range emissions {
		emission := &emissions[idx]
		switch emission.EventType {
		case geofenceEventEntry:
			sawEntry = true
		case geofenceEventExit:
			sawExit = true
		default:
			t.Fatalf("unexpected geofence event type %q", emission.EventType)
		}

		address, ok := anyAsMap(emission.Data["address"])
		if !ok {
			t.Fatalf("expected address payload, got %T", emission.Data["address"])
		}
		if got := stringValue(address, "id"); got != "addr-geo" {
			t.Fatalf("expected address addr-geo, got %q", got)
		}
		circle, ok := anyAsMap(nestedAny(Record(address), "geofence", "circle"))
		if !ok {
			t.Fatal("expected geofence circle payload")
		}
		if got := floatFromAny(circle["radiusMeters"]); got != 4000 {
			t.Fatalf("expected radiusMeters 4000, got %v", got)
		}

		vehicle, ok := anyAsMap(emission.Data["vehicle"])
		if !ok {
			t.Fatalf("expected vehicle payload, got %T", emission.Data["vehicle"])
		}
		if got := stringValue(vehicle, "id"); got != "veh-geo" {
			t.Fatalf("expected vehicle veh-geo, got %q", got)
		}
		if got := stringValue(vehicle, "assetType"); got != "vehicle" {
			t.Fatalf("expected assetType vehicle, got %q", got)
		}
		if got := stringValue(vehicle, "vin"); got != "1FUJGLDR5CLBP9999" {
			t.Fatalf("expected fixture vin, got %q", got)
		}
		if got := stringValue(vehicle, "licensePlate"); got != "TX-GEO1" {
			t.Fatalf("expected fixture license plate, got %q", got)
		}
	}
	if !sawEntry || !sawExit {
		t.Fatalf("expected both entry and exit transitions, entry=%v exit=%v", sawEntry, sawExit)
	}
}

func TestGeofenceWebhookEmissionsAreDeterministic(t *testing.T) {
	t.Parallel()

	live := newGeofenceLiveSimulator()
	now := time.Now().UTC().Truncate(time.Minute)
	first := live.GeofenceWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	second := live.GeofenceWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)

	if len(first) != len(second) {
		t.Fatalf("expected identical emission counts, got %d and %d", len(first), len(second))
	}
	for idx := range first {
		if first[idx].UniqueKey != second[idx].UniqueKey {
			t.Fatalf(
				"expected deterministic emissions, got %q and %q",
				first[idx].UniqueKey,
				second[idx].UniqueKey,
			)
		}
	}
}

func TestServerDispatchGeofenceEventsCountsAndDedupes(t *testing.T) {
	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Simulation.FleetSize = 1
	cfg.Simulation.TripHoursMin = 1
	cfg.Simulation.TripHoursMax = 1
	cfg.Webhooks.Enabled = false
	scriptPath := filepath.Join(t.TempDir(), "script.yaml")
	if err := os.WriteFile(
		scriptPath,
		[]byte("version: 1\ntimezone: UTC\nscenarios: []\n"),
		0o600,
	); err != nil {
		t.Fatalf("write script fixture: %v", err)
	}
	cfg.Simulation.ScriptPath = scriptPath

	store := NewStore(newGeofenceFixture())
	scenarios, err := NewScenarioEngine("geofence-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	srv := NewServer(&cfg, store, scenarios, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats/feed?types=gps", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")

	srv.dispatchGeofenceEvents(request, srv.simNow())
	srv.clock.Step(70 * time.Minute)
	at := srv.simNow()
	srv.dispatchGeofenceEvents(request, at)

	entries := srv.geofenceEntries.Load()
	exits := srv.geofenceExits.Load()
	if entries == 0 || exits == 0 {
		t.Fatalf("expected geofence counters after full loop, entries=%d exits=%d", entries, exits)
	}

	srv.geofenceWindow.reset()
	srv.dispatchGeofenceEvents(request, at)

	if got := srv.geofenceEntries.Load(); got != entries {
		t.Fatalf("expected deduplicated entry count %d, got %d", entries, got)
	}
	if got := srv.geofenceExits.Load(); got != exits {
		t.Fatalf("expected deduplicated exit count %d, got %d", exits, got)
	}
}
