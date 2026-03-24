package sim

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func TestDispatcherEmitsDuplicateDeliveriesWithMetadata(t *testing.T) {
	t.Parallel()

	type deliveryCapture struct {
		DeliveryID string
		Sequence   string
		Attempt    string
		Timestamp  string
		Signature  string
	}
	var mu sync.Mutex
	captures := make([]deliveryCapture, 0, 4)
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			mu.Lock()
			captures = append(captures, deliveryCapture{
				DeliveryID: request.Header.Get("X-Samsara-Sim-Delivery-Id"),
				Sequence:   request.Header.Get("X-Samsara-Sim-Delivery-Sequence"),
				Attempt:    request.Header.Get("X-Samsara-Sim-Delivery-Attempt"),
				Timestamp:  request.Header.Get("X-Samsara-Timestamp"),
				Signature:  request.Header.Get("X-Samsara-Signature"),
			})
			mu.Unlock()
			writer.WriteHeader(http.StatusNoContent)
		}),
	)
	defer server.Close()

	store := NewStore(&Fixture{
		Webhooks: []Record{
			{
				"id":        "wh-1",
				"name":      "realtime sink",
				"url":       server.URL,
				"secretKey": "target-secret",
				"eventTypes": []any{
					"VehicleSpeeding",
				},
				"simDelivery": map[string]any{
					"allowDuplicates":      true,
					"duplicateRate":        1.0,
					"maxDuplicates":        1,
					"allowReorder":         true,
					"reorderWindowMs":      150,
					"allowTimestampSkew":   true,
					"timestampSkewSeconds": 120,
					"retryJitterMs":        0,
				},
			},
		},
	})
	dispatcher := NewDispatcher(config.WebhooksConfig{
		Enabled:        true,
		SigningSecret:  "fallback-secret",
		MaxAttempts:    1,
		InitialBackoff: 10 * time.Millisecond,
	}, store, nil)
	defer dispatcher.Shutdown()

	if err := dispatcher.Dispatch("default", "VehicleSpeeding", map[string]any{
		"id": "evt-speed-1",
		"vehicle": map[string]any{
			"id": "veh-1",
		},
	}); err != nil {
		t.Fatalf("dispatch webhook: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		count := len(captures)
		mu.Unlock()
		if count >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for duplicate deliveries; got %d", count)
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	got := append([]deliveryCapture{}, captures...)
	mu.Unlock()

	if len(got) < 2 {
		t.Fatalf("expected at least 2 deliveries, got %d", len(got))
	}
	sequenceSeen := map[string]struct{}{}
	for _, capture := range got {
		if capture.DeliveryID == "" {
			t.Fatal("expected X-Samsara-Sim-Delivery-Id header")
		}
		if capture.Attempt != "1" {
			t.Fatalf("expected attempt header 1, got %q", capture.Attempt)
		}
		if capture.Timestamp == "" {
			t.Fatal("expected X-Samsara-Timestamp header")
		}
		if capture.Signature == "" {
			t.Fatal("expected X-Samsara-Signature header")
		}
		sequenceSeen[capture.Sequence] = struct{}{}
	}
	if _, ok := sequenceSeen["0"]; !ok {
		t.Fatal("expected base delivery sequence 0")
	}
	if _, ok := sequenceSeen["1"]; !ok {
		t.Fatal("expected duplicate delivery sequence 1")
	}
}

func TestDeliveryOptionsForTargetAppliesOverrides(t *testing.T) {
	t.Parallel()

	options := deliveryOptionsForTarget(&WebhookTarget{
		SimDelivery: map[string]any{
			"allowDuplicates":      false,
			"duplicateRate":        0.5,
			"maxDuplicates":        3,
			"allowReorder":         false,
			"reorderWindowMs":      2500,
			"allowTimestampSkew":   false,
			"timestampSkewSeconds": 300,
			"retryJitterMs":        900,
		},
	})
	if options.AllowDuplicates {
		t.Fatal("expected allowDuplicates override to disable duplicates")
	}
	if options.DuplicateRate != 0.5 {
		t.Fatalf("expected duplicateRate 0.5, got %f", options.DuplicateRate)
	}
	if options.MaxDuplicates != 3 {
		t.Fatalf("expected maxDuplicates 3, got %d", options.MaxDuplicates)
	}
	if options.AllowReorder {
		t.Fatal("expected allowReorder override false")
	}
	if options.ReorderMaxDelay != 2500*time.Millisecond {
		t.Fatalf("expected reorder delay 2500ms, got %s", options.ReorderMaxDelay)
	}
	if options.AllowTimestampSkew {
		t.Fatal("expected allowTimestampSkew override false")
	}
	if options.TimestampSkewMax != 300*time.Second {
		t.Fatalf("expected skew max 300s, got %s", options.TimestampSkewMax)
	}
	if options.RetryJitterMax != 900*time.Millisecond {
		t.Fatalf("expected retry jitter 900ms, got %s", options.RetryJitterMax)
	}
}
