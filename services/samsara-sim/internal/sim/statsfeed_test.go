package sim

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func TestServerVehicleStatsFeedRequiresTypes(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/vehicles/stats/feed")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without types, got %d", response.Code)
	}

	body := mustReadJSONMap(t, response.Body.Bytes())
	if got := stringValue(body, "code"); got != "TYPES_REQUIRED" {
		t.Fatalf("expected TYPES_REQUIRED error code, got %q", got)
	}
}

func TestServerVehicleStatsFeedRejectsUnknownType(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps,notAStat",
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown type, got %d", response.Code)
	}

	body := mustReadJSONMap(t, response.Body.Bytes())
	if got := stringValue(body, "code"); got != "INVALID_TYPES" {
		t.Fatalf("expected INVALID_TYPES error code, got %q", got)
	}
}

func TestServerVehicleStatsFeedCursorPagination(t *testing.T) {
	srv := newEventTestServer(t, "")

	first := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps,engineStates,fuelPercents,obdOdometerMeters",
	)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200 from initial feed call, got %d", first.Code)
	}

	firstPayload := mustReadJSONMap(t, first.Body.Bytes())
	firstRecords := mustReadDataRecords(t, first.Body.Bytes())
	if len(firstRecords) == 0 {
		t.Fatal("expected initial feed call to return vehicle records")
	}
	vehicle := findRecordByID(t, firstRecords, "veh-1")
	assertFeedSampleCount(t, vehicle, "gps", 1)
	assertFeedSampleCount(t, vehicle, "engineStates", 1)
	assertFeedSampleCount(t, vehicle, "fuelPercents", 1)
	assertFeedSampleCount(t, vehicle, "obdOdometerMeters", 1)

	gpsSample := firstFeedSample(t, vehicle, "gps")
	if _, ok := gpsSample["latitude"].(float64); !ok {
		t.Fatalf("expected gps sample latitude, got %v", gpsSample["latitude"])
	}
	if isEcu, ok := gpsSample["isEcuSpeed"].(bool); !ok || isEcu {
		t.Fatalf("expected gps sample isEcuSpeed false, got %v", gpsSample["isEcuSpeed"])
	}

	firstPagination := mustReadPaginationMap(t, firstPayload)
	cursor := stringValue(firstPagination, "endCursor")
	if cursor == "" {
		t.Fatal("expected non-empty endCursor from initial feed call")
	}
	if hasNext, ok := firstPagination["hasNextPage"].(bool); !ok || hasNext {
		t.Fatalf("expected hasNextPage false on initial call, got %v", firstPagination["hasNextPage"])
	}
	cursorTime, _, err := decodeStatsFeedCursor(cursor)
	if err != nil {
		t.Fatalf("expected decodable endCursor, got error: %v", err)
	}

	srv.clock.Step(10 * time.Minute)
	second := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&after="+url.QueryEscape(cursor),
	)
	if second.Code != http.StatusOK {
		t.Fatalf("expected 200 from cursored feed call, got %d", second.Code)
	}

	secondPayload := mustReadJSONMap(t, second.Body.Bytes())
	secondRecords := mustReadDataRecords(t, second.Body.Bytes())
	vehicle = findRecordByID(t, secondRecords, "veh-1")
	samples := feedSamples(t, vehicle, "gps")
	if len(samples) < 4 || len(samples) > 7 {
		t.Fatalf("expected roughly five 2-minute samples after 10m step, got %d", len(samples))
	}
	for _, sample := range samples {
		sampleTime, parseErr := time.Parse(time.RFC3339, stringValue(Record(sample), "time"))
		if parseErr != nil {
			t.Fatalf("failed to parse gps sample time: %v", parseErr)
		}
		if !sampleTime.After(cursorTime) {
			t.Fatalf("expected samples strictly after cursor %s, got %s", cursorTime, sampleTime)
		}
	}

	secondPagination := mustReadPaginationMap(t, secondPayload)
	secondCursor := stringValue(secondPagination, "endCursor")
	if secondCursor == "" || secondCursor == cursor {
		t.Fatalf("expected advanced endCursor, got %q", secondCursor)
	}
	if hasNext, ok := secondPagination["hasNextPage"].(bool); !ok || hasNext {
		t.Fatalf(
			"expected hasNextPage false when caught up, got %v",
			secondPagination["hasNextPage"],
		)
	}

	third := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&after="+url.QueryEscape(secondCursor),
	)
	if third.Code != http.StatusOK {
		t.Fatalf("expected 200 from caught-up feed call, got %d", third.Code)
	}
	thirdPayload := mustReadJSONMap(t, third.Body.Bytes())
	if records := mustReadDataRecords(t, third.Body.Bytes()); len(records) != 0 {
		t.Fatalf("expected no records when caught up, got %d", len(records))
	}
	thirdPagination := mustReadPaginationMap(t, thirdPayload)
	if got := stringValue(thirdPagination, "endCursor"); got != secondCursor {
		t.Fatalf("expected stable cursor when caught up, got %q", got)
	}
}

func TestServerVehicleStatsFeedRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/feed?types=gps&after=not-a-cursor",
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid cursor, got %d", response.Code)
	}

	body := mustReadJSONMap(t, response.Body.Bytes())
	if got := stringValue(body, "code"); got != "INVALID_CURSOR" {
		t.Fatalf("expected INVALID_CURSOR error code, got %q", got)
	}
}

func TestServerVehicleStatsHistoryRequiresWindow(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/vehicles/stats/history?types=gps",
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without time range, got %d", response.Code)
	}

	body := mustReadJSONMap(t, response.Body.Bytes())
	if got := stringValue(body, "code"); got != "TIME_RANGE_REQUIRED" {
		t.Fatalf("expected TIME_RANGE_REQUIRED error code, got %q", got)
	}
}

func TestServerVehicleStatsHistoryReturnsWindowSamples(t *testing.T) {
	srv := newEventTestServer(t, "")
	now := srv.simNow()
	target := fmt.Sprintf(
		"/fleet/vehicles/stats/history?types=gps,batteryMilliVolts,ecuSpeedMph&startTime=%s&endTime=%s",
		url.QueryEscape(now.Add(-10*time.Minute).Format(time.RFC3339)),
		url.QueryEscape(now.Format(time.RFC3339)),
	)
	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from history endpoint, got %d", response.Code)
	}

	records := mustReadDataRecords(t, response.Body.Bytes())
	vehicle := findRecordByID(t, records, "veh-1")
	if got := len(feedSamples(t, vehicle, "gps")); got < 2 {
		t.Fatalf("expected multiple gps history samples, got %d", got)
	}
	if got := len(feedSamples(t, vehicle, "batteryMilliVolts")); got < 2 {
		t.Fatalf("expected multiple battery history samples, got %d", got)
	}
	if got := len(feedSamples(t, vehicle, "ecuSpeedMph")); got < 2 {
		t.Fatalf("expected multiple ecu speed history samples, got %d", got)
	}
}

func mustReadJSONMap(t *testing.T, body []byte) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return payload
}

func mustReadPaginationMap(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()

	pagination, ok := anyAsMap(payload["pagination"])
	if !ok {
		t.Fatalf("expected pagination object, got %T", payload["pagination"])
	}
	return pagination
}

func findRecordByID(t *testing.T, records []map[string]any, wantID string) map[string]any {
	t.Helper()

	for _, record := range records {
		if stringValue(record, "id") == wantID {
			return record
		}
	}
	t.Fatalf("expected record with id %q", wantID)
	return nil
}

func feedSamples(t *testing.T, record map[string]any, statType string) []map[string]any {
	t.Helper()

	raw, ok := record[statType].([]any)
	if !ok {
		t.Fatalf("expected %s array, got %T", statType, record[statType])
	}
	samples := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		sample, sampleOK := anyAsMap(item)
		if !sampleOK {
			t.Fatalf("expected %s sample object, got %T", statType, item)
		}
		samples = append(samples, sample)
	}
	return samples
}

func assertFeedSampleCount(t *testing.T, record map[string]any, statType string, want int) {
	t.Helper()

	if got := len(feedSamples(t, record, statType)); got != want {
		t.Fatalf("expected %d %s samples, got %d", want, statType, got)
	}
}

func firstFeedSample(t *testing.T, record map[string]any, statType string) map[string]any {
	t.Helper()

	samples := feedSamples(t, record, statType)
	if len(samples) == 0 {
		t.Fatalf("expected at least one %s sample", statType)
	}
	return samples[0]
}
