package sim

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFleetSnapshotCachesWaypointsAcrossMutations(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	first := srv.live.loadAssetWaypoints()
	second := srv.live.loadAssetWaypoints()
	if reflect.ValueOf(first).Pointer() != reflect.ValueOf(second).Pointer() {
		t.Fatal("expected repeated reads to share one parsed waypoint map")
	}
	snapshot := srv.live.fleet()
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/drivers/"+fixtureDriverAlex,
		map[string]any{"notes": "cache"},
		http.StatusOK,
	)
	rebuilt := srv.live.fleet()
	if rebuilt == snapshot {
		t.Fatal("expected a mutation to rebuild the fleet snapshot")
	}
	if reflect.ValueOf(rebuilt.waypoints).Pointer() != reflect.ValueOf(first).Pointer() ||
		rebuilt.geometries[fixtureTruck1001] != snapshot.geometries[fixtureTruck1001] {
		t.Fatal("expected unchanged asset locations not to be re-parsed")
	}
	if stringValue(rebuilt.driverByID[fixtureDriverAlex], "notes") != "cache" {
		t.Fatal("expected the rebuilt snapshot to see the mutation")
	}
	if err := srv.store.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if reflect.ValueOf(srv.live.fleet().waypoints).Pointer() == reflect.ValueOf(first).Pointer() {
		t.Fatal("expected a reset to re-read asset locations")
	}
}

func TestFleetResourcesPersistAndFallBackToFixtures(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	srv := newFleetTestServer(t, fleetServerOptions{})
	enableTestPersistence(t, srv.store, path)
	created := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/tags",
		map[string]any{"name": "Persisted Tag"},
		http.StatusOK,
	).data(t)
	requireStatus(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": fixtureDriverCameron, "vehicleId": fixtureTruck1001,
	}, http.StatusCreated)
	waitForStateWrites(t, srv.store, 1)
	if err := srv.store.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	restored := loadDefaultFixtureStore(t)
	enableTestPersistence(t, restored, path)
	tags, err := restored.List(ResourceTags)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if _, ok := idSetFromRecords(tags)[recordID(created)]; !ok {
		t.Fatal("expected the created tag to survive a restart")
	}
	assignments, err := restored.List(ResourceDriverVehicleAssignments)
	if err != nil || len(assignments) != 1 {
		t.Fatalf("expected the API assignment to survive a restart, got %v (%v)", assignments, err)
	}
	next, err := restored.CreateFromAPI(
		ResourceTags,
		Record{"name": "After Restart"},
		CreateOptions{},
	)
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if mustNumericID(t, "tag", recordID(next)) <= mustNumericID(t, "tag", recordID(created)) {
		t.Fatal("expected tag IDs to keep counting up after a restart")
	}

	legacy := filepath.Join(t.TempDir(), "legacy.json")
	content := `{"version":1,"savedAt":"2026-03-01T00:00:00Z","revision":3,"seedFingerprint":"x",` +
		`"counters":{},"state":{"drivers":[{"id":"1654973","name":"Alex Rivera"}]}}`
	if err = os.WriteFile(legacy, []byte(content), 0o600); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}
	upgraded := loadDefaultFixtureStore(t)
	enableTestPersistence(t, upgraded, legacy)
	drivers, _ := upgraded.List(ResourceDrivers)
	fixtureTags, _ := upgraded.List(ResourceTags)
	workflows, _ := upgraded.List(ResourceDriverWorkflows)
	if len(drivers) != 1 || len(fixtureTags) != 24 || len(workflows) != 6 {
		t.Fatalf(
			"expected persisted drivers to win and missing collections to come from fixtures, got %d/%d/%d",
			len(drivers),
			len(fixtureTags),
			len(workflows),
		)
	}
}

func idSetFromRecords(records []Record) map[string]struct{} {
	return idSet(records)
}

func TestSnapshotStatsUseSimClockDeterministically(t *testing.T) {
	t.Parallel()

	left := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	right := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	time.Sleep(20 * time.Millisecond)
	target := "/fleet/vehicles/stats?types=gps,fuelPercents,obdOdometerMeters"
	leftRows := requireStatus(t, left, http.MethodGet, target, nil, http.StatusOK).list(t)
	rightRows := requireStatus(t, right, http.MethodGet, target, nil, http.StatusOK).list(t)
	for idx := range leftRows {
		for _, key := range []string{"gps", "fuelPercent", "obdOdometerMeters"} {
			if !reflect.DeepEqual(leftRows[idx][key], rightRows[idx][key]) {
				t.Fatalf("expected identical %s for %s at the same sim time, got %v vs %v",
					key, recordID(leftRows[idx]), leftRows[idx][key], rightRows[idx][key])
			}
		}
	}
}
