package sim

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

const (
	routeDatasetTestPath = "../../config/datasets/texas_osm_routes.geojson"

	fixtureTruck1001     = "281474977075805"
	fixtureTruck1002     = "281474977075819"
	fixtureTruck1005     = "281474977075891"
	fixtureTruck1012     = "281474977076045"
	fixtureTrailer2041   = "281474979348595"
	fixtureTrailer2042   = "281474979348612"
	fixtureTrailer2049   = "281474979348731"
	fixtureTrailer2053   = "281474979348799"
	fixtureReefer2042    = "281474980315201"
	fixtureYardSpotter   = "281474980315269"
	fixtureDriverAlex    = "1654973"
	fixtureDriverJordan  = "1654988"
	fixtureDriverCameron = "1655297"
	fixtureAustinYard    = "41226316"
	tagTexasOperations   = "342401"
	tagCentralTexas      = "342412"
	tagAustinTerminal    = "342423"
	tagNorthTexas        = "342445"
	tagDFWTerminal       = "342456"
	tagTractors          = "342555"
	tagReeferTrailers    = "342566"
	tagDryVanTrailers    = "342577"
	tagDriverPrograms    = "342610"
	tagHazmatCertified   = "342621"
)

var fleetTestTime = time.Date(2026, time.March, 4, 15, 0, 0, 0, time.UTC)

type fleetServerOptions struct {
	Dataset    bool
	WebhookURL string
	At         time.Time
}

func newFleetTestServer(t *testing.T, options fleetServerOptions) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.RateLimits.Enabled = false
	cfg.Webhooks.Enabled = false
	cfg.Simulation.ScriptPath = ""
	store := loadDefaultFixtureStore(t)
	if options.Dataset {
		if err := ApplyRouteDataset(store, routeDatasetTestPath); err != nil {
			t.Fatalf("apply route dataset: %v", err)
		}
	}
	var dispatcher *Dispatcher
	if options.WebhookURL != "" {
		cfg.Webhooks.Enabled = true
		cfg.Webhooks.MaxAttempts = 1
		cfg.Webhooks.InitialBackoff = 10 * time.Millisecond
		if err := store.Replace(ResourceWebhooks, []Record{{
			"id":   "524002",
			"name": "fleet sink",
			"url":  options.WebhookURL,
			"simDelivery": map[string]any{
				"allowDuplicates":    false,
				"allowReorder":       false,
				"allowTimestampSkew": false,
				"retryJitterMs":      0,
			},
		}}); err != nil {
			t.Fatalf("replace webhooks: %v", err)
		}
		dispatcher = NewDispatcher(cfg.Webhooks, store, nil)
		t.Cleanup(dispatcher.Shutdown)
	} else if err := store.Replace(ResourceWebhooks, []Record{}); err != nil {
		t.Fatalf("clear webhooks: %v", err)
	}
	scenarios, err := NewScenarioEngine("fleet-test-seed", "default")
	if err != nil {
		t.Fatalf("initialize scenario engine: %v", err)
	}
	srv := NewServer(&cfg, store, scenarios, dispatcher, nil)
	at := options.At
	if at.IsZero() {
		at = fleetTestTime
	}
	srv.clock.SetPaused(true)
	srv.clock.SetTime(at)
	return srv
}

type apiResult struct {
	Status  int
	Body    []byte
	Payload map[string]any
}

func callAPI(t *testing.T, srv *Server, method, target string, body any) apiResult {
	t.Helper()

	var response *httptest.ResponseRecorder
	if body == nil {
		response = performAuthorizedRequest(srv, method, target)
	} else {
		response = performAuthorizedRequestWithBody(srv, method, target, body)
	}
	result := apiResult{Status: response.Code, Body: response.Body.Bytes()}
	if len(result.Body) > 0 {
		payload := map[string]any{}
		if err := sonic.Unmarshal(result.Body, &payload); err == nil {
			result.Payload = payload
		}
	}
	return result
}

func (r apiResult) expect(t *testing.T, status int) apiResult {
	t.Helper()

	if r.Status != status {
		t.Fatalf("expected HTTP %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func (r apiResult) expectError(t *testing.T, status int, fragment string) {
	t.Helper()

	r.expect(t, status)
	message := mustReadAPIError(t, r.Body)
	if fragment != "" && !strings.Contains(strings.ToLower(message), strings.ToLower(fragment)) {
		t.Fatalf("expected error mentioning %q, got %q", fragment, message)
	}
}

func (r apiResult) data(t *testing.T) Record {
	t.Helper()

	data, ok := anyAsMap(r.Payload["data"])
	if !ok {
		t.Fatalf("expected object data, got %s", r.Body)
	}
	return Record(data)
}

func (r apiResult) list(t *testing.T) []Record {
	t.Helper()

	items, ok := r.Payload["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %s", r.Body)
	}
	out := make([]Record, 0, len(items))
	for _, item := range items {
		mapped, isMap := anyAsMap(item)
		if !isMap {
			t.Fatalf("expected object items, got %T", item)
		}
		out = append(out, Record(mapped))
	}
	return out
}

func (r apiResult) pagination(t *testing.T) Record {
	t.Helper()

	pagination, ok := anyAsMap(r.Payload["pagination"])
	if !ok {
		t.Fatalf("expected pagination object, got %s", r.Body)
	}
	return Record(pagination)
}

func listIDs(records []Record) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, recordID(record))
	}
	return out
}

func idSet(records []Record) map[string]struct{} {
	return toStringSet(listIDs(records))
}

func requireRecord(t *testing.T, records []Record, id string) Record {
	t.Helper()

	for _, record := range records {
		if recordID(record) == id {
			return record
		}
	}
	t.Fatalf("expected record %s in %v", id, listIDs(records))
	return nil
}

func tagIDsOf(t *testing.T, record Record) []string {
	t.Helper()

	raw, ok := record["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags array on %s, got %T", recordID(record), record["tags"])
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		mapped, _ := anyAsMap(item)
		out = append(out, stringValue(Record(mapped), "id"))
	}
	return out
}

func query(path string, params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return path + "?" + values.Encode()
}

func mustRFC3339(t *testing.T, raw string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}

func waitForWebhookEvents(
	t *testing.T,
	capture *webhookEventCapture,
	eventType string,
	want int,
) []WebhookEvent {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		matched := make([]WebhookEvent, 0, want)
		seen := map[string]struct{}{}
		for _, event := range capture.snapshot() {
			if event.EventType != eventType {
				continue
			}
			if _, dup := seen[event.EventID]; dup {
				continue
			}
			seen[event.EventID] = struct{}{}
			matched = append(matched, event)
		}
		if len(matched) >= want || time.Now().After(deadline) {
			if len(matched) < want {
				t.Fatalf("expected %d %s webhooks, got %d", want, eventType, len(matched))
			}
			return matched
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func webhookData(t *testing.T, event WebhookEvent) Record {
	t.Helper()

	data, ok := anyAsMap(event.Data)
	if !ok {
		t.Fatalf("expected webhook data object, got %T", event.Data)
	}
	return Record(data)
}

func newWebhookSink(t *testing.T) *webhookEventCapture {
	t.Helper()

	capture := &webhookEventCapture{}
	receiver := httptest.NewServer(capture.handler(t))
	t.Cleanup(receiver.Close)
	capture.url = receiver.URL
	return capture
}

func requireStatus(
	t *testing.T,
	srv *Server,
	method, target string,
	body any,
	status int,
) apiResult {
	t.Helper()

	return callAPI(t, srv, method, target, body).expect(t, status)
}
