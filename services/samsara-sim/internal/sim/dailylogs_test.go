package sim

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func dailyLogsQuery(params map[string]string) string {
	return query("/fleet/hos/daily-logs", params)
}

func TestHOSDailyLogsParameterValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})

	cases := []struct {
		name     string
		params   map[string]string
		fragment string
	}{
		{name: "missing both dates", params: map[string]string{}, fragment: "startDate: is required"},
		{name: "missing end date", params: map[string]string{"startDate": "2026-03-01"}, fragment: "endDate: is required"},
		{
			name:     "malformed start date",
			params:   map[string]string{"startDate": "2026-13-99", "endDate": "2026-03-01"},
			fragment: "YYYY-MM-DD",
		},
		{
			name:     "rfc3339 rejected",
			params:   map[string]string{"startDate": "2026-03-01T00:00:00Z", "endDate": "2026-03-02"},
			fragment: "startDate",
		},
		{
			name:     "end before start",
			params:   map[string]string{"startDate": "2026-03-04", "endDate": "2026-03-01"},
			fragment: "greater than or equal to startDate",
		},
		{
			name: "activation status enum",
			params: map[string]string{
				"startDate":              "2026-03-01",
				"endDate":                "2026-03-02",
				"driverActivationStatus": "retired",
			},
			fragment: "driverActivationStatus",
		},
		{
			name:     "foreign cursor",
			params:   map[string]string{"startDate": "2026-03-01", "endDate": "2026-03-02", "after": "missing"},
			fragment: "after",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodGet, dailyLogsQuery(testCase.params), nil).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}

	callAPI(t, srv, http.MethodGet, dailyLogsQuery(map[string]string{
		"startDate": "2026-03-01",
		"endDate":   "2026-03-01",
		"expand":    "vehicle,unsupported",
		"limit":     "9999",
	}), nil).expect(t, http.StatusOK)
}

func TestHOSDailyLogsUseDriverTimezoneDays(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	records := callAPI(t, srv, http.MethodGet, dailyLogsQuery(map[string]string{
		"startDate": "2026-02-26",
		"endDate":   "2026-03-04",
		"driverIds": fixtureDriverRiley,
	}), nil).expect(t, http.StatusOK).list(t)
	if len(records) != 7 {
		t.Fatalf("expected 7 days for the Denver driver, got %d", len(records))
	}
	if got := stringValue(records[0], "startTime"); got != "2026-03-04T07:00:00Z" {
		t.Fatalf("expected newest day first starting at Denver midnight, got %s", got)
	}
	now := srv.simNow()
	for idx, record := range records {
		if nestedString(record, "driver", "timezone") != "America/Denver" {
			t.Fatalf("expected the driver's own timezone, got %v", record["driver"])
		}
		if _, ok := mapOf(record["driver"])["eldSettings"]; !ok {
			t.Fatalf("expected driver eldSettings, got %v", record["driver"])
		}
		start := mustParseRecordTime(t, record, "startTime")
		end := mustParseRecordTime(t, record, "endTime")
		if end.Sub(start) != 24*time.Hour {
			t.Fatalf("expected a 24-hour log day, got %s", end.Sub(start))
		}
		covered := minTime(end, now).Sub(start).Milliseconds()
		durations := mapOf(record["dutyStatusDurations"])
		sum := int64(0)
		for _, key := range []string{"driveDurationMs", "onDutyDurationMs", "offDutyDurationMs", "sleeperBerthDurationMs"} {
			sum += int64(floatFromAny(durations[key]))
		}
		if sum != covered {
			t.Fatalf("day %d: expected durations to cover %dms, got %dms", idx, covered, sum)
		}
		if floatFromAny(durations["activeDurationMs"]) !=
			floatFromAny(durations["driveDurationMs"])+floatFromAny(durations["onDutyDurationMs"]) {
			t.Fatalf("expected activeDurationMs = drive + onDuty, got %v", durations)
		}
		if !reflect.DeepEqual(record["dutyStatusDurations"], record["pendingDutyStatusDurations"]) {
			t.Fatal("expected pending durations to match with no pending carrier edits")
		}
		metadata := Record(mapOf(record["logMetaData"]))
		if stringValue(metadata, "homeTerminalName") != "El Paso Terminal" ||
			!strings.Contains(stringValue(metadata, "homeTerminalFormattedAddress"), "El Paso, TX") {
			t.Fatalf("expected the El Paso home terminal, got %v", metadata)
		}
		if names := listOf(metadata["trailerNames"]); len(names) != 1 || stringOf(names[0]) != "Trailer 2046" {
			t.Fatalf("expected the coupled trailer name, got %v", metadata["trailerNames"])
		}
		vehicles := listOf(metadata["vehicles"])
		if len(vehicles) != 1 || len(mapOf(vehicles[0])) != 1 {
			t.Fatalf("expected unexpanded vehicles to carry only an id, got %v", vehicles)
		}
		for _, key := range []string{"carrierName", "carrierFormattedAddress", "shippingDocs"} {
			if stringValue(metadata, key) == "" {
				t.Fatalf("expected logMetaData.%s, got %v", key, metadata)
			}
		}
		if idx == 0 {
			if metadata["isCertified"] != false {
				t.Fatal("expected the day in progress to be uncertified")
			}
		}
		if idx >= 2 && metadata["isCertified"] != true {
			t.Fatalf("expected days older than the certification grace to be certified, got %v", metadata)
		}
	}
}

func TestHOSDailyLogsExpandActivationTagsAndExternalIDs(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	window := map[string]string{"startDate": "2026-03-03", "endDate": "2026-03-04"}

	expanded := callAPI(t, srv, http.MethodGet, dailyLogsQuery(mergeParams(window, map[string]string{
		"driverIds": "workerId:worker-1001",
		"expand":    "vehicle",
	})), nil).expect(t, http.StatusOK).list(t)
	if len(expanded) != 2 || nestedString(expanded[0], "driver", "id") != fixtureDriverAlex {
		t.Fatalf("expected two days for Alex via external ID, got %d", len(expanded))
	}
	vehicle := mapOf(listOf(nestedAny(expanded[0], "logMetaData", "vehicles"))[0])
	for _, key := range []string{"id", "name", "assetType", "externalIds", "licensePlate", "vehicleVin"} {
		if _, ok := vehicle[key]; !ok {
			t.Fatalf("expected expanded vehicle.%s, got %v", key, vehicle)
		}
	}

	tagged := taggedDriverIDs(t, srv, tagAustinTerminal)
	byTag := callAPI(t, srv, http.MethodGet, dailyLogsQuery(mergeParams(window, map[string]string{
		"tagIds": tagAustinTerminal,
	})), nil).expect(t, http.StatusOK).list(t)
	if len(byTag) != 2*len(tagged) {
		t.Fatalf("expected two days per Austin driver, got %d for %v", len(byTag), tagged)
	}

	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverCameron, map[string]any{
		"driverActivationStatus": "deactivated",
	}).expect(t, http.StatusOK)
	active := callAPI(t, srv, http.MethodGet, dailyLogsQuery(window), nil).expect(t, http.StatusOK).list(t)
	for _, record := range active {
		if nestedString(record, "driver", "id") == fixtureDriverCameron {
			t.Fatal("expected deactivated drivers to be excluded by default")
		}
	}
	deactivated := callAPI(t, srv, http.MethodGet, dailyLogsQuery(mergeParams(window, map[string]string{
		"driverActivationStatus": "deactivated",
	})), nil).expect(t, http.StatusOK).list(t)
	if len(deactivated) != 2 || nestedString(deactivated[0], "driver", "id") != fixtureDriverCameron {
		t.Fatalf("expected only the deactivated driver, got %d records", len(deactivated))
	}
}

func TestHOSDailyLogsPaginationAndDeterminism(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	params := map[string]string{"startDate": "2026-01-01", "endDate": "2026-03-04"}
	full := callAPI(t, srv, http.MethodGet, dailyLogsQuery(params), nil).expect(t, http.StatusOK)
	records := full.list(t)
	if len(records) != 512 || full.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a full 512-record page, got %d", len(records))
	}
	collected := append([]Record{}, records...)
	after := stringOf(full.pagination(t)["endCursor"])
	for after != "" {
		page := callAPI(t, srv, http.MethodGet, dailyLogsQuery(mergeParams(params, map[string]string{"after": after})), nil).
			expect(t, http.StatusOK)
		collected = append(collected, page.list(t)...)
		after = stringOf(page.pagination(t)["endCursor"])
	}
	if len(collected) != 12*63 {
		t.Fatalf("expected 63 days for 12 drivers, got %d", len(collected))
	}
	again := callAPI(t, srv, http.MethodGet, dailyLogsQuery(params), nil).expect(t, http.StatusOK).list(t)
	if !reflect.DeepEqual(records, again) {
		t.Fatal("expected identical daily logs for identical sim time")
	}
	previousDriver, previousStart := "", ""
	for _, record := range collected {
		driverID := nestedString(record, "driver", "id")
		start := stringValue(record, "startTime")
		if driverID < previousDriver {
			t.Fatalf("expected drivers in ascending ID order, got %s after %s", driverID, previousDriver)
		}
		if driverID == previousDriver && start >= previousStart {
			t.Fatalf("expected newest day first per driver, got %s after %s", start, previousStart)
		}
		previousDriver, previousStart = driverID, start
	}
}

func TestLiveSimulatorHOSLogsIncludeEntriesOverlappingWindowStart(t *testing.T) {
	t.Parallel()

	simulator := newTestLiveSimulator()
	now := simulator.anchorTime.Add(30 * time.Hour)
	start := now.Add(-time.Hour)
	end := now

	records := simulator.HOSLogs(now, []string{testDriverID}, &start, &end)
	if len(records) != 1 {
		t.Fatalf("expected one HOS log record, got %d", len(records))
	}
	rawLogs, ok := records[0]["hosLogs"].([]any)
	if !ok || len(rawLogs) == 0 {
		t.Fatal("expected at least one entry overlapping the window")
	}
	coversWindowStart := false
	for _, raw := range rawLogs {
		entry := Record(mapOf(raw))
		logStart := mustParseRecordTime(t, entry, "logStartTime")
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
