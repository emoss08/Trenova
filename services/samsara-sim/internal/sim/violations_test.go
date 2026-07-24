package sim

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestServerHOSViolationsMapsSimEvents(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	event := mustFindViolationSimEvent(t, srv)
	expectedType := hosViolationTypeForSimEvent(event.Type)
	if expectedType == "" {
		t.Fatalf("expected samsara violation mapping for %q", event.Type)
	}

	target := fmt.Sprintf(
		"/fleet/hos/violations?driverIds=drv-1&startTime=%s&endTime=%s",
		url.QueryEscape(event.StartsAt.Add(-time.Hour).Format(time.RFC3339)),
		url.QueryEscape(event.EndsAt.Add(time.Hour).Format(time.RFC3339)),
	)
	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from violations endpoint, got %d", response.Code)
	}

	records := mustReadDataRecords(t, response.Body.Bytes())
	violation := findRecordByID(t, records, event.ID)
	if got := stringValue(violation, "type"); got != expectedType {
		t.Fatalf("expected violation type %q, got %q", expectedType, got)
	}
	if got := stringValue(violation, "description"); got == "" {
		t.Fatal("expected violation description")
	}
	if got := floatFromAny(violation["durationMs"]); got <= 0 {
		t.Fatalf("expected positive durationMs, got %v", violation["durationMs"])
	}
	if got := nestedString(violation, "driver", "id"); got != "drv-1" {
		t.Fatalf("expected driver drv-1, got %q", got)
	}
	if got := nestedString(violation, "driver", "name"); got == "" {
		t.Fatal("expected driver name")
	}

	violationStart := mustParseRFC3339(t, stringValue(violation, "violationStartTime"))
	dayStart := mustParseRFC3339(t, nestedString(violation, "day", "startTime"))
	dayEnd := mustParseRFC3339(t, nestedString(violation, "day", "endTime"))
	if violationStart.Before(dayStart) || violationStart.After(dayEnd) {
		t.Fatalf(
			"expected violation start %s within day window [%s, %s]",
			violationStart,
			dayStart,
			dayEnd,
		)
	}
}

func TestServerHOSViolationsHonorsTypeFilter(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	event := mustFindViolationSimEvent(t, srv)
	expectedType := hosViolationTypeForSimEvent(event.Type)

	target := fmt.Sprintf(
		"/fleet/hos/violations?types=%s&startTime=%s&endTime=%s",
		url.QueryEscape(expectedType),
		url.QueryEscape(event.StartsAt.Add(-time.Hour).Format(time.RFC3339)),
		url.QueryEscape(event.EndsAt.Add(time.Hour).Format(time.RFC3339)),
	)
	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from filtered violations endpoint, got %d", response.Code)
	}

	records := mustReadDataRecords(t, response.Body.Bytes())
	if len(records) == 0 {
		t.Fatal("expected filtered violations to include the target event")
	}
	for _, record := range records {
		if got := stringValue(record, "type"); got != expectedType {
			t.Fatalf("expected only %q violations, got %q", expectedType, got)
		}
	}
}

func TestServerHOSViolationsDefaultsToLast24Hours(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/hos/violations")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from default-window violations endpoint, got %d", response.Code)
	}

	payload := mustReadJSONMap(t, response.Body.Bytes())
	if _, ok := anyAsMap(payload["pagination"]); !ok {
		t.Fatalf("expected pagination envelope, got %T", payload["pagination"])
	}
}

func mustFindViolationSimEvent(t *testing.T, srv *Server) *SimEvent {
	t.Helper()

	now := time.Now().UTC()
	window := srv.live.EventsWindow(
		now.Add(-36*time.Hour),
		now.Add(36*time.Hour),
		[]string{"drv-1"},
		nil,
		0,
	)
	for idx := range window {
		if strings.HasPrefix(window[idx].Type, "hos.violation.") {
			return &window[idx]
		}
	}
	t.Fatal("expected at least one simulated HOS violation event")
	return nil
}
