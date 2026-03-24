package sim

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func TestServerEventsWindowAndActiveEndpoints(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	window := srv.live.EventsWindow(
		time.Now().UTC().Add(-36*time.Hour),
		time.Now().UTC().Add(36*time.Hour),
		[]string{"drv-1"},
		[]string{"veh-1"},
		0,
	)
	if len(window) == 0 {
		t.Fatal("expected simulated event window to contain at least one event")
	}
	target := window[0]

	startRaw := target.StartsAt.Add(-20 * time.Minute).UTC().Format(time.RFC3339)
	endRaw := target.EndsAt.Add(20 * time.Minute).UTC().Format(time.RFC3339)
	windowURL := fmt.Sprintf(
		"/_sim/events/window?driverIds=drv-1&vehicleIds=veh-1&startTime=%s&endTime=%s&limit=64",
		url.QueryEscape(startRaw),
		url.QueryEscape(endRaw),
	)
	windowResponse := performAuthorizedRequest(srv, http.MethodGet, windowURL)
	if windowResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 from events window endpoint, got %d", windowResponse.Code)
	}

	windowData := mustReadDataRecords(t, windowResponse.Body.Bytes())
	if len(windowData) == 0 {
		t.Fatal("expected events window endpoint to return at least one event")
	}
	if !containsEventID(windowData, target.ID) {
		t.Fatalf("expected events window response to include %q", target.ID)
	}

	activeAt := target.StartsAt.Add(time.Minute).UTC().Format(time.RFC3339)
	activeURL := fmt.Sprintf(
		"/_sim/events/active?driverIds=drv-1&vehicleIds=veh-1&atTime=%s",
		url.QueryEscape(activeAt),
	)
	activeResponse := performAuthorizedRequest(srv, http.MethodGet, activeURL)
	if activeResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 from active events endpoint, got %d", activeResponse.Code)
	}

	activeData := mustReadDataRecords(t, activeResponse.Body.Bytes())
	if len(activeData) == 0 {
		t.Fatal("expected active events endpoint to return at least one event")
	}
	if !containsEventID(activeData, target.ID) {
		t.Fatalf("expected active events response to include %q", target.ID)
	}
}

func TestServerDispatchLiveEventsDeduplicatesWebhookByEvent(t *testing.T) {
	var deliveryCount atomic.Int64
	delivered := make(chan struct{}, 8)
	webhookReceiver := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		deliveryCount.Add(1)
		select {
		case delivered <- struct{}{}:
		default:
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer webhookReceiver.Close()

	srv := newEventTestServer(t, webhookReceiver.URL)
	events := srv.live.EventsWindow(
		time.Now().UTC().Add(-36*time.Hour),
		time.Now().UTC().Add(36*time.Hour),
		[]string{"drv-1"},
		[]string{"veh-1"},
		0,
	)
	if len(events) == 0 {
		t.Fatal("expected simulated events for webhook dedupe test")
	}

	at := events[0].StartsAt.Add(time.Minute)
	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats?vehicleIds=veh-1", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")

	srv.dispatchLiveEvents(request, at, []string{"veh-1"})
	srv.dispatchLiveEvents(request, at, []string{"veh-1"})

	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}

	time.Sleep(200 * time.Millisecond)
	if got := deliveryCount.Load(); got != 1 {
		t.Fatalf("expected exactly one deduplicated delivery, got %d", got)
	}
}

func TestServerDispatchLiveEventsHonorsWebhookFaultDrop(t *testing.T) {
	var deliveryCount atomic.Int64
	webhookReceiver := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		deliveryCount.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer webhookReceiver.Close()

	srv := newEventTestServer(t, webhookReceiver.URL)
	if _, err := srv.faults.Add(&FaultRule{
		ID:      "webhook-drop",
		Enabled: true,
		Target: FaultTarget{
			Kind:             "webhook",
			WebhookEventType: "*",
		},
		Match:  FaultMatch{Profile: "default"},
		Effect: FaultEffect{Drop: true},
		Rate:   1,
	}); err != nil {
		t.Fatalf("add webhook fault rule: %v", err)
	}

	events := srv.live.EventsWindow(
		time.Now().UTC().Add(-36*time.Hour),
		time.Now().UTC().Add(36*time.Hour),
		[]string{"drv-1"},
		[]string{"veh-1"},
		0,
	)
	if len(events) == 0 {
		t.Fatal("expected simulated events for webhook drop test")
	}

	at := events[0].StartsAt.Add(time.Minute)
	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats?vehicleIds=veh-1", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	srv.dispatchLiveEvents(request, at, []string{"veh-1"})

	time.Sleep(200 * time.Millisecond)
	if got := deliveryCount.Load(); got != 0 {
		t.Fatalf("expected dropped webhook delivery count 0, got %d", got)
	}
}

func newEventTestServer(t *testing.T, webhookURL string) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token", "dev-samsara-token-readonly"}
	cfg.Simulation.FleetSize = 1
	cfg.Simulation.TripHoursMin = 8
	cfg.Simulation.TripHoursMax = 8
	cfg.Simulation.EventIntensity = "driving"
	cfg.Simulation.ViolationRate = 1
	cfg.Simulation.SpeedingRate = 1
	scriptPath := filepath.Join(t.TempDir(), "script.yaml")
	scriptContent := "version: 1\ntimezone: UTC\nscenarios: []\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o600); err != nil {
		t.Fatalf("write script fixture: %v", err)
	}
	cfg.Simulation.ScriptPath = scriptPath
	cfg.Simulation.ScriptMode = "merge"
	cfg.Simulation.ScriptTimezone = "UTC"

	fixture := &Fixture{
		Assets: []Record{
			{
				"id":   "veh-1",
				"name": "Truck 1001",
				"type": "vehicle",
			},
		},
		AssetLocation: []Record{
			{
				"asset": map[string]any{"id": "veh-1"},
				"location": map[string]any{
					"latitude":       30.2672,
					"longitude":      -97.7431,
					"headingDegrees": 85.0,
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 17.5,
				},
				"happenedAtTime": "2026-03-01T14:00:00Z",
			},
			{
				"asset": map[string]any{"id": "veh-1"},
				"location": map[string]any{
					"latitude":       30.3001,
					"longitude":      -97.7004,
					"headingDegrees": 92.0,
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": 22.3,
				},
				"happenedAtTime": "2026-03-01T14:08:00Z",
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
	if webhookURL != "" {
		cfg.Webhooks.Enabled = true
		cfg.Webhooks.MaxAttempts = 1
		cfg.Webhooks.InitialBackoff = 10 * time.Millisecond
		fixture.Webhooks = []Record{
			{
				"id":   "wh-test",
				"name": "event sink",
				"url":  webhookURL,
				"simDelivery": map[string]any{
					"allowDuplicates":    false,
					"allowReorder":       false,
					"allowTimestampSkew": false,
					"retryJitterMs":      0,
				},
			},
		}
	} else {
		cfg.Webhooks.Enabled = false
	}

	store := NewStore(fixture)
	scenarios, err := NewScenarioEngine("server-events-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}

	var dispatcher *Dispatcher
	if cfg.Webhooks.Enabled {
		dispatcher = NewDispatcher(cfg.Webhooks, store, nil)
		t.Cleanup(dispatcher.Shutdown)
	}

	return NewServer(&cfg, store, scenarios, dispatcher, nil)
}

func performAuthorizedRequest(srv *Server, method, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	response := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(response, request)
	return response
}

func mustReadDataRecords(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	rawData, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("expected response data array, got %T", payload["data"])
	}

	records := make([]map[string]any, 0, len(rawData))
	for _, item := range rawData {
		record, isRecord := anyAsMap(item)
		if isRecord {
			records = append(records, record)
		}
	}
	return records
}

func containsEventID(records []map[string]any, wantID string) bool {
	for _, record := range records {
		if stringValue(record, "id") == wantID {
			return true
		}
	}
	return false
}
