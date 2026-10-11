package sim

import (
	"testing"
	"time"
)

func TestPolygonContainsAndCentroid(t *testing.T) {
	square := []geofenceVertex{
		{Latitude: 29.99, Longitude: -97.01},
		{Latitude: 29.99, Longitude: -96.99},
		{Latitude: 30.01, Longitude: -96.99},
		{Latitude: 30.01, Longitude: -97.01},
	}
	if !polygonContains(square, 30.0, -97.0) {
		t.Fatal("expected the center inside the square")
	}
	if polygonContains(square, 30.05, -97.0) || polygonContains(square, 30.0, -96.9) {
		t.Fatal("expected points outside the square to be excluded")
	}
	center := polygonCentroid(square)
	if haversineMeters(center.Latitude, center.Longitude, 30.0, -97.0) > 1 {
		t.Fatalf("expected the square's centroid at its center, got %v", center)
	}
	shape := geofenceCircle{Vertices: square, Latitude: 30.0, Longitude: -97.0, RadiusMeters: 1}
	if !shape.contains(30.009, -97.009) {
		t.Fatal("expected polygon containment to ignore the bounding radius")
	}
}

func TestGeofenceWebhookEmissionsSupportPolygons(t *testing.T) {
	t.Parallel()

	fixture := newGeofenceFixture()
	fixture.Addresses[0]["geofence"] = map[string]any{
		"polygon": map[string]any{"vertices": []any{
			map[string]any{"latitude": 29.97, "longitude": -97.04},
			map[string]any{"latitude": 29.97, "longitude": -96.96},
			map[string]any{"latitude": 30.03, "longitude": -96.96},
			map[string]any{"latitude": 30.03, "longitude": -97.04},
		}},
	}
	live := NewLiveSimulator(NewStore(fixture), "geo-seed", LiveSimulationOptions{
		FleetSize:    1,
		TripHoursMin: 1,
		TripHoursMax: 1,
	})
	now := stopRichSimTime()
	emissions := live.GeofenceWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	sawEntry, sawExit := false, false
	for idx := range emissions {
		switch emissions[idx].EventType {
		case geofenceEventEntry:
			sawEntry = true
		case geofenceEventExit:
			sawExit = true
		}
		address := mapOf(emissions[idx].Data["address"])
		polygon := mapOf(mapOf(address["geofence"])["polygon"])
		if len(listOf(polygon["vertices"])) != 4 {
			t.Fatalf("expected the polygon in the geofence payload, got %v", address["geofence"])
		}
	}
	if !sawEntry || !sawExit {
		t.Fatalf("expected polygon entry and exit, entry=%v exit=%v", sawEntry, sawExit)
	}

	snapshot := live.fleet()
	if fence := snapshot.geofenceAt(30.0, -97.0); fence == nil || fence.AddressID != "41230101" {
		t.Fatalf("expected the polygon to resolve the address inside it, got %v", fence)
	}
	if fence := snapshot.geofenceAt(30.05, -97.0); fence != nil {
		t.Fatalf("expected no address outside the polygon, got %v", fence.AddressID)
	}
}
