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
		"/fleet/hos/violations?driverIds=1654973&startTime=%s&endTime=%s",
		url.QueryEscape(event.StartsAt.Add(-time.Hour).Format(time.RFC3339)),
		url.QueryEscape(event.EndsAt.Add(time.Hour).Format(time.RFC3339)),
	)
	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from violations endpoint, got %d", response.Code)
	}

	records := mustReadHOSViolations(t, response.Body.Bytes())
	violation := findHOSViolation(t, records, testDriverID, event.StartsAt)
	if got := stringValue(violation, "type"); got != expectedType {
		t.Fatalf("expected violation type %q, got %q", expectedType, got)
	}
	if got := stringValue(violation, "description"); got == "" {
		t.Fatal("expected violation description")
	}
	if got := floatFromAny(violation["durationMs"]); got <= 0 {
		t.Fatalf("expected positive durationMs, got %v", violation["durationMs"])
	}
	if got := nestedString(violation, "driver", "id"); got != testDriverID {
		t.Fatalf("expected driver 1654973, got %q", got)
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

	records := mustReadHOSViolations(t, response.Body.Bytes())
	if len(records) == 0 {
		t.Fatal("expected filtered violations to include the target event")
	}
	for _, record := range records {
		if got := stringValue(record, "type"); got != expectedType {
			t.Fatalf("expected only %q violations, got %q", expectedType, got)
		}
	}
}

func TestServerHOSViolationsDefaultWindowIsNow(t *testing.T) {
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
	now := srv.simNow().UTC().Format(time.RFC3339)
	for _, violation := range mustReadHOSViolations(t, response.Body.Bytes()) {
		if got := stringValue(violation, "violationStartTime"); got != now {
			t.Fatalf("expected only violations starting now (%s), got %s", now, got)
		}
	}
}

func TestServerHOSViolationsFilterOnViolationStartTime(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	event := mustFindViolationSimEvent(t, srv)
	query := func(start, end time.Time) []map[string]any {
		target := fmt.Sprintf(
			"/fleet/hos/violations?driverIds=%s&startTime=%s&endTime=%s",
			testDriverID,
			url.QueryEscape(start.UTC().Format(time.RFC3339)),
			url.QueryEscape(end.UTC().Format(time.RFC3339)),
		)
		response := performAuthorizedRequest(srv, http.MethodGet, target)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200 from violations endpoint, got %d", response.Code)
		}
		return mustReadHOSViolations(t, response.Body.Bytes())
	}
	wantStart := event.StartsAt.UTC().Format(time.RFC3339)
	contains := func(records []map[string]any) bool {
		for _, record := range records {
			if nestedString(record, "driver", "id") == testDriverID &&
				stringValue(record, "violationStartTime") == wantStart {
				return true
			}
		}
		return false
	}

	if !contains(query(event.StartsAt, event.StartsAt)) {
		t.Fatal("expected a window starting and ending at violationStartTime to include it")
	}
	if !contains(query(event.StartsAt.Add(-time.Hour), event.StartsAt)) {
		t.Fatal("expected endTime equal to violationStartTime to include it")
	}
	if contains(query(event.StartsAt.Add(time.Second), event.EndsAt.Add(time.Hour))) {
		t.Fatal("expected a window starting after violationStartTime to exclude it")
	}
	if contains(query(event.StartsAt.Add(-2*time.Hour), event.StartsAt.Add(-time.Second))) {
		t.Fatal("expected a window ending before violationStartTime to exclude it")
	}
}

func TestServerHOSViolationsGroupsByDriverWithoutIDs(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	now := stopRichSimTime()
	target := fmt.Sprintf(
		"/fleet/hos/violations?startTime=%s&endTime=%s",
		url.QueryEscape(now.Add(-7*24*time.Hour).Format(time.RFC3339)),
		url.QueryEscape(now.Add(36*time.Hour).Format(time.RFC3339)),
	)
	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from violations endpoint, got %d", response.Code)
	}

	groups := mustReadDataRecords(t, response.Body.Bytes())
	if len(groups) == 0 {
		t.Fatal("expected at least one violation group")
	}
	seenDrivers := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if len(group) != 1 {
			t.Fatalf("expected group to hold only violations, got keys %v", group)
		}
		rawViolations, ok := group["violations"].([]any)
		if !ok || len(rawViolations) == 0 {
			t.Fatalf("expected non-empty violations array, got %T", group["violations"])
		}
		groupDriver := ""
		for _, raw := range rawViolations {
			violation, okMap := anyAsMap(raw)
			if !okMap {
				t.Fatalf("expected violation object, got %T", raw)
			}
			if _, hasID := violation["id"]; hasID {
				t.Fatal("expected violation without id field")
			}
			driverID := nestedString(violation, "driver", "id")
			if groupDriver == "" {
				groupDriver = driverID
			}
			if driverID != groupDriver {
				t.Fatalf("expected one driver per group, got %q and %q", groupDriver, driverID)
			}
		}
		if _, dup := seenDrivers[groupDriver]; dup {
			t.Fatalf("expected driver %q in a single group", groupDriver)
		}
		seenDrivers[groupDriver] = struct{}{}
	}
}

func mustReadHOSViolations(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	groups := mustReadDataRecords(t, body)
	out := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		rawViolations, ok := group["violations"].([]any)
		if !ok {
			t.Fatalf("expected violations array in group, got %T", group["violations"])
		}
		for _, raw := range rawViolations {
			violation, okMap := anyAsMap(raw)
			if !okMap {
				t.Fatalf("expected violation object, got %T", raw)
			}
			out = append(out, violation)
		}
	}
	return out
}

func findHOSViolation(
	t *testing.T,
	violations []map[string]any,
	driverID string,
	startsAt time.Time,
) map[string]any {
	t.Helper()

	wantStart := startsAt.UTC().Format(time.RFC3339)
	for _, violation := range violations {
		if nestedString(violation, "driver", "id") == driverID &&
			stringValue(violation, "violationStartTime") == wantStart {
			return violation
		}
	}
	t.Fatalf("expected violation for %s starting %s", driverID, wantStart)
	return nil
}

func mustFindViolationSimEvent(t *testing.T, srv *Server) *SimEvent {
	t.Helper()

	now := stopRichSimTime()
	window := srv.live.EventsWindow(
		now.Add(-36*time.Hour),
		now.Add(36*time.Hour),
		[]string{testDriverID},
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
