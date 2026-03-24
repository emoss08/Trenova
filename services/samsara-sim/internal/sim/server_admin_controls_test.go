package sim

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func TestServerTimeControlEndpoints(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")

	getResp := performAuthorizedRequestWithBody(srv, http.MethodGet, "/_sim/time", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from time get, got %d", getResp.Code)
	}

	setBody := map[string]any{
		"paused":  true,
		"speed":   2.0,
		"setTime": "2026-03-02T08:00:00Z",
	}
	setResp := performAuthorizedRequestWithBody(srv, http.MethodPut, "/_sim/time", setBody)
	if setResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from time put, got %d", setResp.Code)
	}
	setData := mustReadDataMap(t, setResp.Body.Bytes())
	if paused, _ := setData["paused"].(bool); !paused {
		t.Fatal("expected paused=true after put")
	}

	stepResp := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/_sim/time/step",
		map[string]any{
			"durationMs": 60000,
		},
	)
	if stepResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from time step, got %d", stepResp.Code)
	}
	stepData := mustReadDataMap(t, stepResp.Body.Bytes())
	nowRaw, _ := stepData["now"].(string)
	parsed, err := time.Parse(time.RFC3339, nowRaw)
	if err != nil {
		t.Fatalf("parse now after step: %v", err)
	}
	if parsed.UTC().Format(time.RFC3339) != "2026-03-02T08:01:00Z" {
		t.Fatalf("expected stepped time 08:01Z, got %s", parsed.UTC().Format(time.RFC3339))
	}
}

func TestServerFaultInjectionEndpoints(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	createRuleBody := map[string]any{
		"enabled": true,
		"target": map[string]any{
			"kind":        "endpoint",
			"method":      "GET",
			"pathPattern": "/fleet/vehicles/stats",
		},
		"match": map[string]any{
			"profile": "default",
		},
		"effect": map[string]any{
			"statusCode": 429,
		},
		"rate": 1.0,
	}

	createResp := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/_sim/faults/rules",
		createRuleBody,
	)
	if createResp.Code != http.StatusOK {
		t.Fatalf("expected 200 creating fault rule, got %d", createResp.Code)
	}

	faulted := performAuthorizedRequestWithBody(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?vehicleIds=veh-1",
		nil,
	)
	if faulted.Code != http.StatusTooManyRequests {
		t.Fatalf("expected injected 429 for vehicle stats, got %d", faulted.Code)
	}

	resetResp := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/_sim/faults/reset",
		map[string]any{},
	)
	if resetResp.Code != http.StatusOK {
		t.Fatalf("expected 200 resetting faults, got %d", resetResp.Code)
	}

	recovered := performAuthorizedRequestWithBody(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats?vehicleIds=veh-1",
		nil,
	)
	if recovered.Code != http.StatusOK {
		t.Fatalf("expected healthy 200 after fault reset, got %d", recovered.Code)
	}
}

func TestServerWebhookInboxCaptureAndList(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	webhookBody := map[string]any{
		"eventType": "VehicleSpeeding",
		"eventTime": "2026-03-02T12:00:00Z",
		"data": map[string]any{
			"id": "evt-1",
		},
	}
	encoded, err := sonic.Marshal(webhookBody)
	if err != nil {
		t.Fatalf("marshal webhook body: %v", err)
	}

	captureRequest := httptest.NewRequest(
		http.MethodPost,
		"/_sim/webhooks/inbox",
		bytes.NewReader(encoded),
	)
	captureRequest.Header.Set("Content-Type", "application/json")
	captureRequest.Header.Set("X-Samsara-Sim-Delivery-Id", "dlv-1")
	captureRequest.Header.Set("X-Samsara-Sim-Delivery-Sequence", "0")
	captureRequest.Header.Set("X-Samsara-Sim-Delivery-Attempt", "1")
	captureRequest.Header.Set("X-Samsara-Timestamp", "1772452800")
	captureResponse := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(captureResponse, captureRequest)
	if captureResponse.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from webhook capture, got %d", captureResponse.Code)
	}

	listResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodGet,
		"/_sim/webhooks/inbox?limit=10",
		nil,
	)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook inbox list, got %d", listResponse.Code)
	}
	records := mustReadDataRecordsForAdminTest(t, listResponse.Body.Bytes())
	if len(records) == 0 {
		t.Fatal("expected at least one captured webhook record")
	}
	first := records[0]
	if got := stringValue(Record(first), "eventType"); got != "VehicleSpeeding" {
		t.Fatalf("expected eventType VehicleSpeeding, got %q", got)
	}
	delivery, ok := anyAsMap(first["delivery"])
	if !ok {
		t.Fatal("expected delivery object in inbox payload")
	}
	if stringValue(Record(delivery), "id") != "dlv-1" {
		t.Fatalf("expected delivery id dlv-1, got %q", stringValue(Record(delivery), "id"))
	}
}

func performAuthorizedRequestWithBody(
	srv *Server,
	method string,
	target string,
	body any,
) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		encoded, err := sonic.Marshal(body)
		if err == nil {
			payload = encoded
		}
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(response, request)
	return response
}

func mustReadDataMap(t *testing.T, body []byte) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object data payload, got %T", payload["data"])
	}
	return data
}

func mustReadDataRecordsForAdminTest(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	rawData, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("expected array data payload, got %T", payload["data"])
	}
	out := make([]map[string]any, 0, len(rawData))
	for _, item := range rawData {
		record, okRecord := anyAsMap(item)
		if okRecord {
			out = append(out, record)
		}
	}
	return out
}
