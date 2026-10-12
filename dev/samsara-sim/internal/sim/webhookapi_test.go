package sim

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func TestWebhookEventTypeEnumIsSortedAndComplete(t *testing.T) {
	if !slices.IsSorted(webhookEventTypes) {
		t.Fatal("expected the event-type enum to stay sorted for binary search")
	}
	if len(webhookEventTypes) != 35 {
		t.Fatalf("expected the 35 spec event types, got %d", len(webhookEventTypes))
	}
	for _, eventType := range []string{
		"ShipmentTrackingEvent", "VisualSearchMatch", "WorkOrderCreatedOrChanged", "DriverUpdated",
	} {
		if !isWebhookEventType(eventType) {
			t.Fatalf("expected %s in the enum", eventType)
		}
	}
	if isWebhookEventType("VehicleSpeeding") {
		t.Fatal("expected unknown event types to be rejected")
	}
}

func TestWebhookCreateDefaultsAndShape(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	created := requireStatus(t, srv, http.MethodPost, "/webhooks", map[string]any{
		"name":      "TMS events",
		"url":       "https://tms.example.com/samsara/webhooks",
		"secretKey": "client-chosen",
		"id":        "1",
	}, http.StatusOK)
	if _, wrapped := created.Payload["data"]; wrapped {
		t.Fatalf("expected the create response to be the bare webhook object, got %s", created.Body)
	}
	webhook := Record(created.Payload)
	id := recordID(webhook)
	if id == "1" || webhook["version"] != "2018-01-01" {
		t.Fatalf("expected a server ID and the default version, got %v", webhook)
	}
	if secret := stringValue(webhook, "secretKey"); secret != generatedWebhookSecret(id) ||
		secret == "client-chosen" {
		t.Fatalf("expected a generated secretKey, got %q", secret)
	}
	if len(listOf(webhook["eventTypes"])) != 0 || len(listOf(webhook["customHeaders"])) != 0 {
		t.Fatalf("expected empty eventTypes and customHeaders, got %v", webhook)
	}
	for key := range webhook {
		if !slices.Contains(
			[]string{"id", "name", "url", "version", "secretKey", "eventTypes", "customHeaders"},
			key,
		) {
			t.Fatalf("unexpected webhook field %q", key)
		}
	}

	fetched := requireStatus(t, srv, http.MethodGet, "/webhooks/"+id, nil, http.StatusOK)
	if fetched.Payload["secretKey"] != webhook["secretKey"] ||
		fetched.Payload["name"] != "TMS events" {
		t.Fatalf("expected GET to return the bare webhook, got %s", fetched.Body)
	}
	requireStatus(t, srv, http.MethodGet, "/webhooks/999", nil, http.StatusNotFound)
}

func TestWebhookValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	base := func(mutate func(body map[string]any)) map[string]any {
		body := map[string]any{"name": "Hook", "url": "https://hooks.example.com/in"}
		mutate(body)
		return body
	}
	header := func(key, value string) map[string]any {
		return map[string]any{"key": key, "value": value}
	}
	sixHeaders := make([]any, 0, 6)
	for idx := 0; idx < 6; idx++ {
		sixHeaders = append(sixHeaders, header("X-Custom-"+string(rune('A'+idx)), "v"))
	}
	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{"missing name", base(func(b map[string]any) { delete(b, "name") }), "name is required"},
		{"missing url", base(func(b map[string]any) { delete(b, "url") }), "url is required"},
		{
			"long name",
			base(func(b map[string]any) { b["name"] = strings.Repeat("n", 256) }),
			"at most 255",
		},
		{"relative url", base(func(b map[string]any) { b["url"] = "/hook" }), "absolute http"},
		{
			"ftp url",
			base(func(b map[string]any) { b["url"] = "ftp://hooks.example.com" }),
			"absolute http",
		},
		{
			"long url",
			base(
				func(b map[string]any) { b["url"] = "https://h.example.com/" + strings.Repeat("a", 2030) },
			),
			"at most 2047",
		},
		{"version", base(func(b map[string]any) { b["version"] = "2020-01-01" }), "version"},
		{
			"event type",
			base(func(b map[string]any) { b["eventTypes"] = []any{"VehicleSpeeding"} }),
			"eventTypes",
		},
		{
			"too many headers",
			base(func(b map[string]any) { b["customHeaders"] = sixHeaders }),
			"at most 5",
		},
		{"header key", base(func(b map[string]any) {
			b["customHeaders"] = []any{header("Bad Key", "v")}
		}), "header name"},
		{"header value", base(func(b map[string]any) {
			b["customHeaders"] = []any{header("X-Key", strings.Repeat("v", 101))}
		}), "at most 100"},
		{"header newline", base(func(b map[string]any) {
			b["customHeaders"] = []any{header("X-Key", "a\r\nb")}
		}), "line breaks"},
		{"header missing value", base(func(b map[string]any) {
			b["customHeaders"] = []any{map[string]any{"key": "X-Key"}}
		}), "value is required"},
		{"duplicate header", base(func(b map[string]any) {
			b["customHeaders"] = []any{header("X-Key", "1"), header("x-key", "2")}
		}), "duplicates"},
		{"long header key", base(func(b map[string]any) {
			b["customHeaders"] = []any{header(strings.Repeat("k", 101), "v")}
		}), "at most 100"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/webhooks", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
	listed := requireStatus(t, srv, http.MethodGet, "/webhooks", nil, http.StatusOK).list(t)
	if len(listed) != 0 {
		t.Fatalf("expected no webhooks after failed creates, got %d", len(listed))
	}
}

func TestWebhookPatchListAndDelete(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	ids := make([]string, 0, 3)
	for _, name := range []string{"Alpha", "Bravo", "Charlie"} {
		created := requireStatus(t, srv, http.MethodPost, "/webhooks", map[string]any{
			"name":       name,
			"url":        "https://hooks.example.com/" + strings.ToLower(name),
			"version":    "2024-02-27",
			"eventTypes": []any{"DriverUpdated", "AddressCreated"},
			"customHeaders": []any{
				map[string]any{"key": "X-Api-Key", "value": "k-" + name},
			},
		}, http.StatusOK)
		ids = append(ids, recordID(Record(created.Payload)))
	}

	page := requireStatus(t, srv, http.MethodGet, "/webhooks?limit=2", nil, http.StatusOK)
	if len(page.list(t)) != 2 || page.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a 2-webhook page, got %s", page.Body)
	}
	filtered := requireStatus(t, srv, http.MethodGet,
		"/webhooks?ids="+ids[0]+","+ids[2], nil, http.StatusOK).list(t)
	if strings.Join(listIDs(filtered), ",") != ids[0]+","+ids[2] {
		t.Fatalf("expected the ids filter to select two webhooks, got %v", listIDs(filtered))
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/webhooks?limit=513",
		nil,
	).expectError(t, http.StatusBadRequest, "limit")

	patched := requireStatus(t, srv, http.MethodPatch, "/webhooks/"+ids[1], map[string]any{
		"url":           "https://hooks.example.com/bravo-v2",
		"eventTypes":    []any{"RouteStopArrival"},
		"customHeaders": nil,
		"secretKey":     "ignored",
	}, http.StatusOK)
	webhook := Record(patched.Payload)
	if webhook["url"] != "https://hooks.example.com/bravo-v2" || webhook["name"] != "Bravo" ||
		webhook["version"] != "2024-02-27" {
		t.Fatalf("expected a partial update, got %v", webhook)
	}
	if types := stringListValues(
		webhook["eventTypes"],
	); len(types) != 1 ||
		types[0] != "RouteStopArrival" {
		t.Fatalf("expected eventTypes replaced, got %v", webhook["eventTypes"])
	}
	if len(listOf(webhook["customHeaders"])) != 0 ||
		webhook["secretKey"] != generatedWebhookSecret(ids[1]) {
		t.Fatalf("expected headers cleared and secretKey kept, got %v", webhook)
	}
	callAPI(t, srv, http.MethodPatch, "/webhooks/"+ids[1], map[string]any{"name": nil}).
		expectError(t, http.StatusBadRequest, "name cannot be null")
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/webhooks/999",
		map[string]any{"name": "x"},
		http.StatusNotFound,
	)

	requireStatus(t, srv, http.MethodDelete, "/webhooks/"+ids[0], nil, http.StatusNoContent)
	requireStatus(t, srv, http.MethodDelete, "/webhooks/"+ids[0], nil, http.StatusNotFound)
	requireStatus(t, srv, http.MethodGet, "/webhooks/"+ids[0], nil, http.StatusNotFound)
}

type headerCapture struct {
	mu       sync.Mutex
	requests []http.Header
	types    []string
}

func (c *headerCapture) handler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		c.mu.Lock()
		c.requests = append(c.requests, request.Header.Clone())
		c.types = append(c.types, request.Header.Get("X-Samsara-Event-Type"))
		c.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (c *headerCapture) snapshot() ([]http.Header, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]http.Header{}, c.requests...), append([]string{}, c.types...)
}

func TestDispatcherDeliversSubscribedTypesWithCustomHeaders(t *testing.T) {
	capture := &headerCapture{}
	receiver := httptest.NewServer(capture.handler())
	t.Cleanup(receiver.Close)

	cfg := config.Default().Webhooks
	cfg.Enabled = true
	cfg.MaxAttempts = 1
	cfg.InitialBackoff = 10 * time.Millisecond
	noJitter := map[string]any{
		"allowDuplicates":    false,
		"allowReorder":       false,
		"allowTimestampSkew": false,
		"retryJitterMs":      0,
	}
	store := NewStore(&Fixture{Webhooks: []Record{
		{
			"id":            "524100",
			"name":          "subscribed",
			"url":           receiver.URL + "/subscribed",
			"eventTypes":    []any{"AddressCreated"},
			"customHeaders": []any{map[string]any{"key": "X-Tenant", "value": "tenant-7"}},
			"simDelivery":   noJitter,
		},
		{
			"id":          "524107",
			"name":        "no subscriptions",
			"url":         receiver.URL + "/empty",
			"eventTypes":  []any{},
			"simDelivery": noJitter,
		},
	}})
	dispatcher := NewDispatcher(cfg, store, nil)
	t.Cleanup(dispatcher.Shutdown)

	if err := dispatcher.Dispatch("default", "AddressCreated", map[string]any{
		"address": map[string]any{"id": "41226316"},
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := dispatcher.Dispatch("default", "DriverUpdated", map[string]any{
		"driver": map[string]any{"id": "1654973"},
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := dispatcher.Dispatch("default", "VehicleSpeeding", map[string]any{}); err == nil {
		t.Fatal("expected an unknown event type to be refused")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		headers, types := capture.snapshot()
		if len(headers) >= 1 || time.Now().After(deadline) {
			if len(headers) != 1 || types[0] != "AddressCreated" {
				t.Fatalf("expected exactly one AddressCreated delivery, got %v", types)
			}
			if headers[0].Get("X-Tenant") != "tenant-7" {
				t.Fatalf("expected the custom header on the delivery, got %v", headers[0])
			}
			if headers[0].Get("Content-Type") != "application/json" {
				t.Fatalf("expected JSON content type, got %v", headers[0])
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if headers, _ := capture.snapshot(); len(headers) != 1 {
		t.Fatalf(
			"expected the unsubscribed webhook and event type to stay silent, got %d",
			len(headers),
		)
	}
}

func TestFixtureWebhookSubscribesToDriverUpdated(t *testing.T) {
	store := loadDefaultFixtureStore(t)
	targets := store.WebhookTargets("DriverUpdated")
	if len(targets) != 1 || targets[0].ID != "523918" {
		t.Fatalf("expected the fixture webhook to receive DriverUpdated, got %v", targets)
	}
}
