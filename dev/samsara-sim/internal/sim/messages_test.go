package sim

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMessageListWindowValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	for _, target := range []string{
		"/v1/fleet/messages?endMs=yesterday",
		"/v1/fleet/messages?endMs=1772460000000.5",
		"/v1/fleet/messages?durationMs=1h",
		"/v1/fleet/messages?durationMs=-1",
		"/v1/fleet/messages?endMs=-5",
	} {
		callAPI(t, srv, http.MethodGet, target, nil).expectError(t, http.StatusBadRequest, "parameter")
	}

	window := requireStatus(
		t, srv, http.MethodGet,
		"/v1/fleet/messages?endMs=1772374300000&durationMs=1800000", nil, http.StatusOK,
	).list(t)
	fixtureWindow := make([]Record, 0, len(window))
	for _, message := range window {
		if nestedString(message, "sender", "type") == "dispatch" {
			fixtureWindow = append(fixtureWindow, message)
		}
	}
	if len(fixtureWindow) != 2 {
		t.Fatalf("expected the two fixture dispatch messages, got %d", len(fixtureWindow))
	}
	first, _ := int64Value(fixtureWindow[0]["sentAtMs"])
	second, _ := int64Value(fixtureWindow[1]["sentAtMs"])
	if first < second {
		t.Fatal("expected messages from most recently sent to least recently sent")
	}
	for _, message := range fixtureWindow {
		for _, key := range []string{"driverId", "isRead", "sender", "sentAtMs", "text"} {
			if _, ok := message[key]; !ok {
				t.Fatalf("expected %s on every message, got %v", key, message)
			}
		}
	}

	defaults := requireStatus(t, srv, http.MethodGet, "/v1/fleet/messages", nil, http.StatusOK).list(t)
	if len(defaults) == 0 {
		t.Fatal("expected driver messages in the default 24 hour window")
	}
	cutoff := srv.simNow().Add(-24 * time.Hour).UnixMilli()
	sawDriver := false
	for _, message := range defaults {
		sentAt, _ := int64Value(message["sentAtMs"])
		if sentAt < cutoff || sentAt > srv.simNow().UnixMilli() {
			t.Fatalf("expected messages inside the default window, got %d", sentAt)
		}
		if nestedString(message, "sender", "type") == "driver" {
			sawDriver = true
			if nestedString(message, "sender", "name") == "" {
				t.Fatalf("expected the driver's name as sender, got %v", message["sender"])
			}
		}
	}
	if !sawDriver {
		t.Fatal("expected driver-sent messages from the simulation")
	}
	empty := requireStatus(
		t, srv, http.MethodGet, "/v1/fleet/messages?durationMs=0&endMs=1", nil, http.StatusOK,
	).list(t)
	if len(empty) != 0 {
		t.Fatalf("expected no messages at the epoch, got %d", len(empty))
	}
}

func TestMessageCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverJordan, map[string]any{
		"driverActivationStatus": "deactivated",
	}).expect(t, http.StatusOK)

	alex := float64(1654973)
	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{"missing text", map[string]any{"driverIds": []any{alex}}, "text is required"},
		{"blank text", map[string]any{"driverIds": []any{alex}, "text": "  "}, "text must not be empty"},
		{"long text", map[string]any{"driverIds": []any{alex}, "text": strings.Repeat("x", 2501)}, "2500"},
		{"text type", map[string]any{"driverIds": []any{alex}, "text": 5}, "text must be a string"},
		{"missing ids", map[string]any{"text": "hi"}, "driverIds is required"},
		{"empty ids", map[string]any{"text": "hi", "driverIds": []any{}}, "at least one"},
		{"ids type", map[string]any{"text": "hi", "driverIds": "1654973"}, "array"},
		{"string id", map[string]any{"text": "hi", "driverIds": []any{"1654973"}}, "driverIds[0]"},
		{"decimal id", map[string]any{"text": "hi", "driverIds": []any{1654973.5}}, "driverIds[0]"},
		{"negative id", map[string]any{"text": "hi", "driverIds": []any{alex, float64(-3)}}, "driverIds[1]"},
		{"unknown id", map[string]any{"text": "hi", "driverIds": []any{alex, float64(42)}}, "42"},
		{"deactivated", map[string]any{"text": "hi", "driverIds": []any{float64(1654988)}}, "1654988"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/v1/fleet/messages", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
	before := requireStatus(t, srv, http.MethodGet, "/v1/fleet/messages?durationMs=60000", nil, http.StatusOK).list(t)
	for _, message := range before {
		if nestedString(message, "sender", "type") == "dispatch" {
			t.Fatalf("expected rejected requests to store nothing, got %v", message)
		}
	}
}

func TestMessageCreateStoresAndBecomesRead(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	created := requireStatus(t, srv, http.MethodPost, "/v1/fleet/messages", map[string]any{
		"driverIds": []any{float64(1654973), float64(1654973), float64(1655297)},
		"text":      "Call dispatch when the trailer is loaded.",
	}, http.StatusOK).list(t)
	if len(created) != 2 {
		t.Fatalf("expected one message per unique driver, got %d", len(created))
	}
	if id, _ := int64Value(created[0]["driverId"]); id != 1654973 {
		t.Fatalf("expected the first driver echoed as an integer, got %v", created[0]["driverId"])
	}

	dispatchMessages := func() []Record {
		out := []Record{}
		for _, message := range requireStatus(
			t, srv, http.MethodGet, "/v1/fleet/messages?durationMs=3600000", nil, http.StatusOK,
		).list(t) {
			if nestedString(message, "sender", "type") == "dispatch" {
				out = append(out, message)
			}
		}
		return out
	}
	stored := dispatchMessages()
	if len(stored) != 2 {
		t.Fatalf("expected both stored dispatch messages, got %d", len(stored))
	}
	for _, message := range stored {
		if message["isRead"] != false {
			t.Fatalf("expected a just-sent message to be unread, got %v", message)
		}
		if sentAt, _ := int64Value(message["sentAtMs"]); sentAt != srv.simNow().UnixMilli() {
			t.Fatalf("expected sentAtMs at sim now, got %d", sentAt)
		}
	}
	srv.clock.SetTime(srv.simNow().Add(messageReadDelayMin + messageReadDelaySpread))
	for _, message := range dispatchMessages() {
		if message["isRead"] != true {
			t.Fatalf("expected drivers to read messages within the read delay, got %v", message)
		}
	}
	for _, message := range stored {
		driverID, _ := int64Value(message["driverId"])
		if strconv.FormatInt(driverID, 10) != fixtureDriverAlex && driverID != 1655297 {
			t.Fatalf("unexpected driver %d", driverID)
		}
	}
}
