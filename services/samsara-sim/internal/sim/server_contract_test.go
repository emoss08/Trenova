package sim

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
)

func TestServerRejectsWriteOnReadOnlyToken(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	body := map[string]any{"name": "Read Only Address"}
	encoded, err := sonic.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewReader(encoded))
	request.Header.Set("Authorization", "Bearer dev-samsara-token-readonly")
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for read-only token write, got %d", response.Code)
	}
	if code := mustReadErrorCode(t, response.Body.Bytes()); code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN code, got %q", code)
	}
}

func TestServerStrictPaginationValidation(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	invalidLimit := performAuthorizedRequest(srv, http.MethodGet, "/assets?limit=0")
	if invalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid limit, got %d", invalidLimit.Code)
	}
	if code := mustReadErrorCode(t, invalidLimit.Body.Bytes()); code != "INVALID_LIMIT" {
		t.Fatalf("expected INVALID_LIMIT code, got %q", code)
	}

	invalidCursor := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/assets?limit=5&after=missing-cursor",
	)
	if invalidCursor.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid cursor, got %d", invalidCursor.Code)
	}
	if code := mustReadErrorCode(t, invalidCursor.Body.Bytes()); code != "INVALID_CURSOR" {
		t.Fatalf("expected INVALID_CURSOR code, got %q", code)
	}
}

func TestServerRateLimitReturns429WithHeaders(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	var response *httptest.ResponseRecorder
	for idx := 0; idx < 240; idx++ {
		response = performAuthorizedRequest(srv, http.MethodGet, "/assets")
		if response.Code == http.StatusTooManyRequests {
			break
		}
	}
	if response == nil || response.Code != http.StatusTooManyRequests {
		t.Fatal("expected rate limiter to return 429 within request burst")
	}
	if response.Header().Get("X-RateLimit-Limit") == "" {
		t.Fatal("expected X-RateLimit-Limit header on 429 response")
	}
	if response.Header().Get("X-RateLimit-Remaining") == "" {
		t.Fatal("expected X-RateLimit-Remaining header on 429 response")
	}
	if response.Header().Get("X-RateLimit-Reset") == "" {
		t.Fatal("expected X-RateLimit-Reset header on 429 response")
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
}

func TestServerCreateConflictReturns409(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	body := map[string]any{
		"id":   "veh-1",
		"name": "Truck 1001 duplicate",
		"type": "vehicle",
	}
	response := performAuthorizedRequestWithBody(srv, http.MethodPost, "/assets", body)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate record create, got %d", response.Code)
	}
	if code := mustReadErrorCode(t, response.Body.Bytes()); code != "CONFLICT" {
		t.Fatalf("expected CONFLICT code, got %q", code)
	}
}

func TestServerRouteListIncludesLifecycleStops(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/routes?ids=route-1")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 for route list, got %d", response.Code)
	}

	records := mustReadDataRecords(t, response.Body.Bytes())
	if len(records) != 1 {
		t.Fatalf("expected one route record, got %d", len(records))
	}
	route := records[0]
	if stringValue(Record(route), "status") == "" {
		t.Fatal("expected dynamic route status in route list")
	}
	rawStops, ok := route["stops"].([]any)
	if !ok || len(rawStops) < 3 {
		t.Fatalf("expected >=3 lifecycle stops, got %d", len(rawStops))
	}
	firstStop, ok := rawStops[0].(map[string]any)
	if !ok {
		t.Fatal("expected stop object payload")
	}
	if stringValue(Record(firstStop), "etaTime") == "" {
		t.Fatal("expected stop etaTime in lifecycle payload")
	}
}

func mustReadErrorCode(t *testing.T, body []byte) string {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	value, _ := payload["code"].(string)
	return value
}
