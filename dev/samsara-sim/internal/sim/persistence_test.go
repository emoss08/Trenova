package sim

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

const testFlushDelay = 20 * time.Millisecond

func encodeForComparison(value any) (string, error) {
	encoded, err := sonic.ConfigStd.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func newPersistenceFixture() *Fixture {
	return &Fixture{
		Addresses: []Record{
			{"id": "41226316", "name": "Austin Main Yard"},
			{"id": "41226329", "name": "Dallas Yard"},
		},
		Drivers:  []Record{{"id": testDriverID, "name": "Alex Rivera"}},
		Webhooks: []Record{{"id": "523918", "name": "TMS", "url": "http://127.0.0.1:1/old"}},
		AssetLocation: []Record{
			{
				"asset":          map[string]any{"id": fixtureVehicleID},
				"happenedAtTime": "2026-01-01T00:00:00Z",
			},
		},
	}
}

func enableTestPersistence(t *testing.T, store *Store, path string) PersistenceStatus {
	t.Helper()

	status, err := store.EnablePersistence(PersistenceOptions{
		Path:       path,
		FlushDelay: testFlushDelay,
	})
	if err != nil {
		t.Fatalf("enable persistence: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.ClosePersistence(); closeErr != nil {
			t.Errorf("close persistence: %v", closeErr)
		}
	})
	return status
}

func waitForStateWrites(t *testing.T, store *Store, want uint64) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if store.currentPersister().writes.Load() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d state writes", want)
}

func temporaryStateFiles(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read state directory: %v", err)
	}
	leftovers := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			leftovers = append(leftovers, entry.Name())
		}
	}
	return leftovers
}

func TestPersistenceRoundTripRestoresStateAndCounters(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "state.json")
	original := NewStore(newPersistenceFixture())
	status := enableTestPersistence(t, original, path)
	if status.Restored {
		t.Fatal("expected a fresh start without a state file")
	}

	at := time.Date(2026, time.February, 2, 8, 0, 0, 0, time.UTC)
	driver, err := original.CreateFromAPI(
		ResourceDrivers,
		Record{"name": "Persisted"},
		CreateOptions{At: at},
	)
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	deleted, err := original.CreateFromAPI(
		ResourceDrivers,
		Record{"name": "Deleted"},
		CreateOptions{At: at},
	)
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	if err = original.Delete(ResourceDrivers, recordID(deleted)); err != nil {
		t.Fatalf("delete driver: %v", err)
	}
	if _, err = original.PatchFromAPI(
		ResourceWebhooks,
		"523918",
		Record{"url": "http://127.0.0.1:1/new"},
		at,
	); err != nil {
		t.Fatalf("patch webhook: %v", err)
	}
	if err = original.Delete(ResourceAddresses, "41226329"); err != nil {
		t.Fatalf("delete address: %v", err)
	}
	if err = original.AppendMessages(
		[]Record{{"driverId": int64(1654973), "text": "hi"}},
	); err != nil {
		t.Fatalf("append message: %v", err)
	}
	if err = original.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	restored := NewStore(newPersistenceFixture())
	restoredStatus := enableTestPersistence(t, restored, path)
	if !restoredStatus.Restored || restoredStatus.SeedChanged || restoredStatus.SavedAt.IsZero() {
		t.Fatalf("expected a restored state with matching seed, got %+v", restoredStatus)
	}

	gotDriver, err := restored.Get(ResourceDrivers, recordID(driver))
	if err != nil {
		t.Fatalf("expected API-created driver to survive restart: %v", err)
	}
	if stringValue(gotDriver, "createdAtTime") != at.Format(time.RFC3339) {
		t.Fatalf("expected persisted createdAtTime, got %v", gotDriver["createdAtTime"])
	}
	if _, err = restored.Get(
		ResourceDrivers,
		recordID(deleted),
	); !errors.Is(
		err,
		ErrRecordNotFound,
	) {
		t.Fatalf("expected deleted driver to stay deleted, got %v", err)
	}
	webhook, err := restored.Get(ResourceWebhooks, "523918")
	if err != nil || stringValue(webhook, "url") != "http://127.0.0.1:1/new" {
		t.Fatalf("expected webhook URL change to persist, got %v %v", webhook, err)
	}
	if _, err = restored.Get(ResourceAddresses, "41226329"); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("expected deleted address to stay deleted, got %v", err)
	}
	messages, err := restored.List(ResourceMessages)
	if err != nil || len(messages) != 1 {
		t.Fatalf("expected the appended message, got %v %v", messages, err)
	}
	locations, err := restored.List(ResourceAssetLocation)
	if err != nil || len(locations) != 1 {
		t.Fatalf(
			"expected transient asset locations to come from the seed, got %d %v",
			len(locations),
			err,
		)
	}

	next, err := restored.CreateFromAPI(
		ResourceDrivers,
		Record{"name": "After restart"},
		CreateOptions{At: at},
	)
	if err != nil {
		t.Fatalf("create after restart: %v", err)
	}
	if mustNumericID(t, "driver", recordID(next)) <= mustNumericID(t, "driver", recordID(deleted)) {
		t.Fatalf(
			"expected ids to continue above the deleted %s, got %s",
			recordID(deleted),
			recordID(next),
		)
	}
}

func TestPersistenceCoalescesBurstsIntoOneWrite(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	store := NewStore(newPersistenceFixture())
	if _, err := store.EnablePersistence(PersistenceOptions{
		Path:       path,
		FlushDelay: 300 * time.Millisecond,
	}); err != nil {
		t.Fatalf("enable persistence: %v", err)
	}
	t.Cleanup(func() {
		if err := store.ClosePersistence(); err != nil {
			t.Errorf("close persistence: %v", err)
		}
	})

	for idx := 0; idx < 200; idx++ {
		if _, err := store.CreateFromAPI(
			ResourceAddresses,
			Record{"name": "Burst"},
			CreateOptions{At: time.Now()},
		); err != nil {
			t.Fatalf("create address: %v", err)
		}
	}
	waitForStateWrites(t, store, 1)
	time.Sleep(400 * time.Millisecond)
	if writes := store.currentPersister().writes.Load(); writes != 1 {
		t.Fatalf("expected one coalesced write for the burst, got %d", writes)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	loaded := persistedState{}
	if err = sonic.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("decode state file: %v", err)
	}
	if loaded.Version != persistedStateVersion || len(loaded.State.Addresses) != 202 {
		t.Fatalf(
			"expected all burst records in the single write, got %d",
			len(loaded.State.Addresses),
		)
	}
	if loaded.State.AssetLocation != nil {
		t.Fatal("expected transient asset locations to be left out of the state file")
	}
	if leftovers := temporaryStateFiles(t, filepath.Dir(path)); len(leftovers) != 0 {
		t.Fatalf("expected no temporary files after a write, got %v", leftovers)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected a private state file, got %v", info.Mode().Perm())
	}
}

func TestWriteFileAtomicKeepsOriginalOnFailure(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "state.json")
	if err := writeFileAtomic(path, []byte(`{"first":true}`)); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := writeFileAtomic(path, []byte(`{"second":true}`)); err != nil {
		t.Fatalf("second write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != `{"second":true}` {
		t.Fatalf("expected the replacement content, got %q %v", raw, err)
	}

	blocked := filepath.Join(directory, "blocked")
	if err = os.MkdirAll(filepath.Join(blocked, "child"), 0o750); err != nil {
		t.Fatalf("create blocking directory: %v", err)
	}
	if err = writeFileAtomic(blocked, []byte(`{"never":true}`)); err == nil {
		t.Fatal("expected replacing a non-empty directory to fail")
	}
	if leftovers := temporaryStateFiles(t, directory); len(leftovers) != 0 {
		t.Fatalf("expected the failed write to clean up, got %v", leftovers)
	}
	raw, err = os.ReadFile(path)
	if err != nil || string(raw) != `{"second":true}` {
		t.Fatalf("expected the existing state to be untouched, got %q %v", raw, err)
	}
}

func TestPersistenceRefusesCorruptState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    error
	}{
		{
			name:    "truncated",
			content: `{"version":1,"state":{"drivers":[{"id":"1"`,
			want:    ErrStateCorrupt,
		},
		{name: "empty", content: "  \n", want: ErrStateCorrupt},
		{name: "not json", content: "garbage", want: ErrStateCorrupt},
		{name: "missing state", content: `{"version":1,"counters":{}}`, want: ErrStateCorrupt},
		{name: "future version", content: `{"version":2,"state":{}}`, want: ErrStateVersion},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "state.json")
			if err := os.WriteFile(path, []byte(testCase.content), 0o600); err != nil {
				t.Fatalf("write corrupt state: %v", err)
			}
			store := NewStore(newPersistenceFixture())
			_, err := store.EnablePersistence(PersistenceOptions{Path: path})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("expected %v, got %v", testCase.want, err)
			}
			if !strings.Contains(err.Error(), path) {
				t.Fatalf("expected the error to name the file, got %v", err)
			}
			if store.currentPersister() != nil {
				t.Fatal("expected persistence to stay disabled")
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil || string(raw) != testCase.content {
				t.Fatal("expected the corrupt file to be left in place for inspection")
			}
		})
	}
}

func TestPersistenceDetectsSeedChanges(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	original := NewStore(newPersistenceFixture())
	enableTestPersistence(t, original, path)
	if _, err := original.CreateFromAPI(
		ResourceDrivers,
		Record{"name": "x"},
		CreateOptions{},
	); err != nil {
		t.Fatalf("create driver: %v", err)
	}
	if err := original.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	changedFixture := newPersistenceFixture()
	changedFixture.Drivers = append(changedFixture.Drivers, Record{"id": "1654999", "name": "New"})
	changed := NewStore(changedFixture)
	status := enableTestPersistence(t, changed, path)
	if !status.Restored || !status.SeedChanged {
		t.Fatalf("expected the seed change to be reported, got %+v", status)
	}
	if _, err := changed.Get(ResourceDrivers, "1654999"); !errors.Is(err, ErrRecordNotFound) {
		t.Fatal("expected persisted records to take precedence over the new seed")
	}
}

func TestPersistenceResetRemovesStateAndRestoresBaseline(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	store := NewStore(newPersistenceFixture())
	enableTestPersistence(t, store, path)
	if err := store.ReplaceBaseline(ResourceAssetLocation, []Record{
		{"asset": map[string]any{"id": fixtureVehicleID}, "happenedAtTime": "2026-01-01T00:00:00Z"},
		{"asset": map[string]any{"id": fixtureVehicleID}, "happenedAtTime": "2026-01-01T00:01:00Z"},
	}); err != nil {
		t.Fatalf("replace baseline: %v", err)
	}
	created, err := store.CreateFromAPI(ResourceDrivers, Record{"name": "Temp"}, CreateOptions{})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	waitForStateWrites(t, store, 1)

	if err = store.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected reset to remove the state file, got %v", statErr)
	}
	if _, err = store.Get(ResourceDrivers, recordID(created)); !errors.Is(err, ErrRecordNotFound) {
		t.Fatal("expected reset to drop API-created records")
	}
	locations, err := store.List(ResourceAssetLocation)
	if err != nil || len(locations) != 2 {
		t.Fatalf("expected reset to keep the dataset baseline, got %d %v", len(locations), err)
	}

	time.Sleep(3 * testFlushDelay)
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("expected no stale flush to recreate the state file after reset")
	}

	if _, err = store.CreateFromAPI(
		ResourceDrivers,
		Record{"name": "After"},
		CreateOptions{},
	); err != nil {
		t.Fatalf("create after reset: %v", err)
	}
	waitForStateWrites(t, store, 2)
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected mutations after reset to persist again, got %v", statErr)
	}

	fresh := NewStore(newPersistenceFixture())
	status := enableTestPersistence(t, fresh, path)
	if !status.Restored {
		t.Fatal("expected the post-reset mutation to be restored")
	}
	drivers, err := fresh.List(ResourceDrivers)
	if err != nil || len(drivers) != 2 {
		t.Fatalf(
			"expected the seed driver plus the post-reset driver, got %d %v",
			len(drivers),
			err,
		)
	}
}

func TestServerStateResetEndpointClearsPersistedState(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	srv := newDefaultFixtureServer(t, func(cfg *config.Config) {
		cfg.Auth.Tokens = []string{"dev-samsara-token"}
	})
	enableTestPersistence(t, srv.store, path)

	response := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/fleet/drivers",
		map[string]any{
			"name":     "Persist Me",
			"username": "persist.me",
			"password": "Sup3rSecret!",
		},
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected driver create to succeed, got %d", response.Code)
	}
	waitForStateWrites(t, srv.store, 1)

	summary := performAuthorizedRequest(srv, http.MethodGet, "/_sim/state/summary")
	data := mustReadDataMap(t, summary.Body.Bytes())
	persistence, ok := anyAsMap(data["persistence"])
	if !ok || stringValue(Record(persistence), "path") != path {
		t.Fatalf("expected persistence details in the summary, got %v", data["persistence"])
	}

	reset := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/_sim/state/reset",
		map[string]any{},
	)
	if reset.Code != http.StatusOK {
		t.Fatalf("expected reset to succeed, got %d", reset.Code)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected reset to remove the persisted state, got %v", err)
	}
}

func TestEnablePersistenceRejectsInvalidUse(t *testing.T) {
	t.Parallel()

	store := NewStore(newPersistenceFixture())
	if _, err := store.EnablePersistence(
		PersistenceOptions{Path: "  "},
	); !errors.Is(
		err,
		ErrStatePathRequired,
	) {
		t.Fatalf("expected ErrStatePathRequired, got %v", err)
	}
	enableTestPersistence(t, store, filepath.Join(t.TempDir(), "state.json"))
	if _, err := store.EnablePersistence(PersistenceOptions{
		Path: filepath.Join(t.TempDir(), "other.json"),
	}); !errors.Is(err, ErrPersistenceEnabled) {
		t.Fatalf("expected ErrPersistenceEnabled, got %v", err)
	}
}
