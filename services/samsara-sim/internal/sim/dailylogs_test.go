package sim

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func TestServerHOSDailyLogsDateParamValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "malformed start date",
			target:     "/fleet/hos/daily-logs?startDate=2026-13-99",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "malformed end date",
			target:     "/fleet/hos/daily-logs?endDate=not-a-date",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "rfc3339 rejected",
			target:     "/fleet/hos/daily-logs?startDate=2026-07-01T00:00:00Z",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "end date before start date",
			target:     "/fleet/hos/daily-logs?startDate=2026-07-10&endDate=2026-07-01",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "range wider than 30 days",
			target:     "/fleet/hos/daily-logs?startDate=2026-01-01&endDate=2026-06-01",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "valid explicit range",
			target:     "/fleet/hos/daily-logs?startDate=2026-07-01&endDate=2026-07-05",
			wantStatus: http.StatusOK,
		},
		{
			name:       "default range",
			target:     "/fleet/hos/daily-logs",
			wantStatus: http.StatusOK,
		},
	}

	srv := newEventTestServer(t, "")
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			response := performAuthorizedRequest(srv, http.MethodGet, testCase.target)
			if response.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d", testCase.wantStatus, response.Code)
			}
			if testCase.wantCode == "" {
				return
			}
			if code := mustReadErrorCode(t, response.Body.Bytes()); code != testCase.wantCode {
				t.Fatalf("expected error code %q, got %q", testCase.wantCode, code)
			}
		})
	}
}

func TestLiveSimulatorHOSDailyLogsDeterministic(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(50 * time.Hour)
	endDate := now.UTC().Truncate(24 * time.Hour)
	startDate := endDate.Add(-3 * 24 * time.Hour)

	first := simulator.HOSDailyLogs(now, []string{"drv-1"}, startDate, endDate)
	second := simulator.HOSDailyLogs(now, []string{"drv-1"}, startDate, endDate)
	if len(first) == 0 {
		t.Fatal("expected daily log records")
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("expected identical daily log output for identical sim time")
	}
}

func TestLiveSimulatorHOSDailyLogsDurationsCoverDayWindow(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(72 * time.Hour)
	endDate := now.UTC().Truncate(24 * time.Hour).Add(-2 * 24 * time.Hour)
	startDate := endDate.Add(-5 * 24 * time.Hour)

	records := simulator.HOSDailyLogs(now, []string{"drv-1"}, startDate, endDate)
	if len(records) != 6 {
		t.Fatalf("expected 6 completed-day records, got %d", len(records))
	}

	sawDriving := false
	for _, record := range records {
		startTime := mustParseRecordTime(t, record, "startTime")
		endTime := mustParseRecordTime(t, record, "endTime")
		windowMs := endTime.Sub(startTime).Milliseconds()
		if windowMs <= 0 {
			t.Fatalf("expected positive day window, got %dms", windowMs)
		}

		driveMs := floatFromAny(nestedAny(record, "dutyStatusDurations", "driveDurationMs"))
		onDutyMs := floatFromAny(nestedAny(record, "dutyStatusDurations", "onDutyDurationMs"))
		offDutyMs := floatFromAny(nestedAny(record, "dutyStatusDurations", "offDutyDurationMs"))
		sleeperMs := floatFromAny(
			nestedAny(record, "dutyStatusDurations", "sleeperBerthDurationMs"),
		)
		activeMs := floatFromAny(nestedAny(record, "dutyStatusDurations", "activeDurationMs"))

		sumMs := int64(driveMs + onDutyMs + offDutyMs + sleeperMs)
		if diff := sumMs - windowMs; diff < -5 || diff > 5 {
			t.Fatalf(
				"expected duty durations (%dms) to cover day window (%dms), diff %dms",
				sumMs,
				windowMs,
				diff,
			)
		}
		if activeMs != driveMs+onDutyMs {
			t.Fatalf("expected activeDurationMs %f, got %f", driveMs+onDutyMs, activeMs)
		}

		durations, ok := anyAsMap(record["dutyStatusDurations"])
		if !ok {
			t.Fatal("expected dutyStatusDurations object")
		}
		pending, ok := anyAsMap(record["pendingDutyStatusDurations"])
		if !ok {
			t.Fatal("expected pendingDutyStatusDurations object")
		}
		if !reflect.DeepEqual(durations, pending) {
			t.Fatal("expected pending durations to mirror duty status durations")
		}

		distanceMeters := floatFromAny(
			nestedAny(record, "distanceTraveled", "driveDistanceMeters"),
		)
		if driveMs > 0 {
			sawDriving = true
			if distanceMeters <= 0 {
				t.Fatalf("expected positive drive distance for %fms driving", driveMs)
			}
		}

		metadata, ok := anyAsMap(record["logMetaData"])
		if !ok {
			t.Fatal("expected logMetaData object")
		}
		if stringValue(Record(metadata), "carrierName") != dailyLogCarrierName {
			t.Fatalf("unexpected carrier name %q", stringValue(Record(metadata), "carrierName"))
		}
		if stringValue(Record(metadata), "homeTerminalName") == "" {
			t.Fatal("expected homeTerminalName to be populated")
		}
		if !strings.HasPrefix(stringValue(Record(metadata), "shippingDocs"), "SD-") {
			t.Fatalf("unexpected shippingDocs %q", stringValue(Record(metadata), "shippingDocs"))
		}
		if now.Sub(endTime) > dailyLogCertifyGrace {
			certified, isBool := metadata["isCertified"].(bool)
			if !isBool || !certified {
				t.Fatalf("expected day older than 24h to be certified, got %v", metadata["isCertified"])
			}
			if stringValue(Record(metadata), "certifiedAtTime") == "" {
				t.Fatal("expected certifiedAtTime on certified day")
			}
		}
		vehicles, ok := metadata["vehicles"].([]any)
		if !ok || len(vehicles) != 1 {
			t.Fatalf("expected one vehicle in logMetaData, got %v", metadata["vehicles"])
		}
		vehicle, ok := anyAsMap(vehicles[0])
		if !ok || stringValue(Record(vehicle), "id") != "veh-1" {
			t.Fatalf("expected assigned vehicle veh-1, got %v", vehicles[0])
		}

		if nestedString(record, "driver", "timezone") != dailyLogDriverTimezone {
			t.Fatalf("unexpected driver timezone %q", nestedString(record, "driver", "timezone"))
		}
	}
	if !sawDriving {
		t.Fatal("expected at least one day with driving duration")
	}
}

func TestLiveSimulatorHOSDailyLogsDriverFilterAndOrder(t *testing.T) {
	t.Parallel()

	fixture := &Fixture{
		Drivers: []Record{
			{"id": "drv-a", "name": "Driver A"},
			{"id": "drv-b", "name": "Driver B"},
		},
	}
	simulator := NewLiveSimulator(NewStore(fixture), "daily-logs-filter-seed")
	now := simulator.anchorTime.Add(48 * time.Hour)
	endDate := now.UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	startDate := endDate.Add(-2 * 24 * time.Hour)

	filtered := simulator.HOSDailyLogs(now, []string{"drv-a"}, startDate, endDate)
	if len(filtered) != 3 {
		t.Fatalf("expected 3 records for filtered driver, got %d", len(filtered))
	}
	for _, record := range filtered {
		if nestedString(record, "driver", "id") != "drv-a" {
			t.Fatalf("unexpected driver id %q", nestedString(record, "driver", "id"))
		}
	}

	all := simulator.HOSDailyLogs(now, nil, startDate, endDate)
	if len(all) != 6 {
		t.Fatalf("expected 6 records for both drivers, got %d", len(all))
	}

	previousDriver := ""
	previousStart := ""
	for _, record := range all {
		driverID := nestedString(record, "driver", "id")
		startTime := stringValue(record, "startTime")
		if driverID < previousDriver {
			t.Fatalf("expected driver ids in ascending order, got %q after %q", driverID, previousDriver)
		}
		if driverID == previousDriver && startTime >= previousStart {
			t.Fatalf(
				"expected newest day first per driver, got %q after %q",
				startTime,
				previousStart,
			)
		}
		previousDriver = driverID
		previousStart = startTime
	}
}

func TestServerHOSDailyLogsPagination(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	first := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/hos/daily-logs?driverIds=drv-1&limit=2",
	)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200 for first page, got %d", first.Code)
	}
	firstRecords, firstPagination := mustReadDailyLogPage(t, first.Body.Bytes())
	if len(firstRecords) != 2 {
		t.Fatalf("expected 2 records on first page, got %d", len(firstRecords))
	}
	if hasNext, ok := firstPagination["hasNextPage"].(bool); !ok || !hasNext {
		t.Fatalf("expected hasNextPage true on first page, got %v", firstPagination["hasNextPage"])
	}
	endCursor, ok := firstPagination["endCursor"].(string)
	if !ok || endCursor == "" {
		t.Fatalf("expected non-empty endCursor, got %v", firstPagination["endCursor"])
	}

	second := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/hos/daily-logs?driverIds=drv-1&limit=2&after="+url.QueryEscape(endCursor),
	)
	if second.Code != http.StatusOK {
		t.Fatalf("expected 200 for second page, got %d", second.Code)
	}
	secondRecords, _ := mustReadDailyLogPage(t, second.Body.Bytes())
	if len(secondRecords) != 2 {
		t.Fatalf("expected 2 records on second page, got %d", len(secondRecords))
	}
	if stringValue(firstRecords[1], "startTime") <= stringValue(secondRecords[0], "startTime") {
		t.Fatal("expected second page to continue with older days")
	}

	invalidCursor := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/hos/daily-logs?driverIds=drv-1&limit=2&after=missing-cursor",
	)
	if invalidCursor.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid cursor, got %d", invalidCursor.Code)
	}
	if code := mustReadErrorCode(t, invalidCursor.Body.Bytes()); code != "INVALID_CURSOR" {
		t.Fatalf("expected INVALID_CURSOR code, got %q", code)
	}
}

func TestLiveSimulatorHOSLogsIncludeEntriesOverlappingWindowStart(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(30 * time.Hour)
	start := now.Add(-time.Hour)
	end := now

	records := simulator.HOSLogs(now, []string{"drv-1"}, &start, &end)
	if len(records) != 1 {
		t.Fatalf("expected one HOS log record, got %d", len(records))
	}
	rawLogs, ok := records[0]["hosLogs"].([]any)
	if !ok {
		t.Fatal("expected hosLogs array")
	}
	if len(rawLogs) == 0 {
		t.Fatal("expected at least one entry overlapping the window")
	}

	coversWindowStart := false
	for _, raw := range rawLogs {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			continue
		}
		logStart, err := time.Parse(time.RFC3339, stringValue(entry, "logStartTime"))
		if err != nil {
			t.Fatalf("failed parsing logStartTime: %v", err)
		}
		if logStart.After(end) {
			t.Fatalf("entry starting at %s is outside the window", logStart.Format(time.RFC3339))
		}
		if !logStart.After(start) {
			coversWindowStart = true
		}
	}
	if !coversWindowStart {
		t.Fatal("expected an entry that started before the window and overlaps into it")
	}
}

func mustReadDailyLogPage(
	t *testing.T,
	body []byte,
) ([]map[string]any, map[string]any) {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	rawData, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("expected response data array, got %T", payload["data"])
	}
	records := make([]map[string]any, 0, len(rawData))
	for _, item := range rawData {
		record, isRecord := anyAsMap(item)
		if isRecord {
			records = append(records, record)
		}
	}
	pagination, ok := anyAsMap(payload["pagination"])
	if !ok {
		t.Fatalf("expected pagination object, got %T", payload["pagination"])
	}
	return records, pagination
}

func mustParseRecordTime(t *testing.T, record Record, key string) time.Time {
	t.Helper()

	raw := stringValue(record, key)
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("failed parsing %s %q: %v", key, raw, err)
	}
	return parsed.UTC()
}
