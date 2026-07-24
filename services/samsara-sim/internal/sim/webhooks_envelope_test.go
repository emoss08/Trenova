package sim

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

var eventIDPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func TestDispatcherEmitsRealSamsaraEnvelopeAndSignature(t *testing.T) {
	t.Parallel()

	type deliveryCapture struct {
		Body      []byte
		Timestamp string
		Signature string
	}
	var mu sync.Mutex
	captures := make([]deliveryCapture, 0, 1)
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, readErr := io.ReadAll(request.Body)
			if readErr != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			mu.Lock()
			captures = append(captures, deliveryCapture{
				Body:      body,
				Timestamp: request.Header.Get("X-Samsara-Timestamp"),
				Signature: request.Header.Get("X-Samsara-Signature"),
			})
			mu.Unlock()
			writer.WriteHeader(http.StatusNoContent)
		}),
	)
	defer server.Close()

	rawKey := "super-secret-key"
	store := NewStore(&Fixture{
		Webhooks: []Record{
			{
				"id":         "wh-real",
				"name":       "real envelope sink",
				"url":        server.URL,
				"secretKey":  base64.StdEncoding.EncodeToString([]byte(rawKey)),
				"eventTypes": []any{"VehicleCreated"},
				"simDelivery": map[string]any{
					"allowDuplicates":    false,
					"allowReorder":       false,
					"allowTimestampSkew": false,
					"retryJitterMs":      0,
				},
			},
		},
	})
	dispatcher := NewDispatcher(config.WebhooksConfig{
		Enabled:        true,
		MaxAttempts:    1,
		InitialBackoff: 10 * time.Millisecond,
	}, store, nil)
	defer dispatcher.Shutdown()

	if err := dispatcher.Dispatch("default", "VehicleCreated", map[string]any{
		"id":   "veh-9",
		"name": "Truck 9",
	}); err != nil {
		t.Fatalf("dispatch webhook: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		count := len(captures)
		mu.Unlock()
		if count >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for webhook delivery")
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	capture := captures[0]
	mu.Unlock()

	envelope := map[string]any{}
	if err := sonic.Unmarshal(capture.Body, &envelope); err != nil {
		t.Fatalf("failed to decode webhook body: %v", err)
	}
	eventID := stringValue(envelope, "eventId")
	if !eventIDPattern.MatchString(eventID) {
		t.Fatalf("expected uuid-like eventId, got %q", eventID)
	}
	if got := stringValue(envelope, "eventType"); got != "VehicleCreated" {
		t.Fatalf("expected eventType VehicleCreated, got %q", got)
	}
	if got := int64(floatFromAny(envelope["orgId"])); got != webhookOrgID {
		t.Fatalf("expected orgId %d, got %d", webhookOrgID, got)
	}
	if got := stringValue(envelope, "webhookId"); got != "wh-real" {
		t.Fatalf("expected webhookId wh-real, got %q", got)
	}
	eventTime := stringValue(envelope, "eventTime")
	if _, err := time.Parse(webhookEventTimeLayout, eventTime); err != nil {
		t.Fatalf("expected millisecond RFC3339 eventTime, got %q: %v", eventTime, err)
	}
	data, ok := anyAsMap(envelope["data"])
	if !ok {
		t.Fatalf("expected data payload, got %T", envelope["data"])
	}
	if got := stringValue(data, "id"); got != "veh-9" {
		t.Fatalf("expected data id veh-9, got %q", got)
	}

	if _, err := time.Parse(time.RFC3339, capture.Timestamp); err != nil {
		t.Fatalf("expected RFC3339 timestamp header, got %q: %v", capture.Timestamp, err)
	}
	mac := hmac.New(sha256.New, []byte(rawKey))
	_, _ = mac.Write([]byte("v1:"))
	_, _ = mac.Write([]byte(capture.Timestamp))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write(capture.Body)
	expected := "v1=" + hex.EncodeToString(mac.Sum(nil))
	if capture.Signature != expected {
		t.Fatalf("expected signature %q, got %q", expected, capture.Signature)
	}
}

func TestWebhookSigningKeyFallsBackToRawBytes(t *testing.T) {
	t.Parallel()

	if got := string(webhookSigningKey("not-base64!")); got != "not-base64!" {
		t.Fatalf("expected raw-byte fallback, got %q", got)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("decoded-key"))
	if got := string(webhookSigningKey(encoded)); got != "decoded-key" {
		t.Fatalf("expected base64-decoded key, got %q", got)
	}
}

func TestServerWebhookCreateGeneratesSecretKey(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedJSONRequest(
		srv,
		http.MethodPost,
		"/webhooks",
		`{"name":"created sink","url":"http://localhost:9/inbox"}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook create, got %d", response.Code)
	}

	created := mustReadJSONMap(t, response.Body.Bytes())
	secret := stringValue(created, "secretKey")
	if secret == "" {
		t.Fatal("expected generated secretKey on created webhook")
	}
	if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
		t.Fatalf("expected base64 secretKey, got %q: %v", secret, err)
	}

	listResponse := performAuthorizedRequest(srv, http.MethodGet, "/webhooks")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook list, got %d", listResponse.Code)
	}
	records := mustReadDataRecords(t, listResponse.Body.Bytes())
	listed := findRecordByID(t, records, stringValue(created, "id"))
	if got := stringValue(listed, "secretKey"); got != secret {
		t.Fatalf("expected listed secretKey %q, got %q", secret, got)
	}
}

func performAuthorizedJSONRequest(
	srv *Server,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(response, request)
	return response
}
