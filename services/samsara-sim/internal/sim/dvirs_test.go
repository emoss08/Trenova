package sim

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func newDvirFixture(driverCount int) *Fixture {
	fixture := &Fixture{}
	for idx := 1; idx <= driverCount; idx++ {
		driverID := fmt.Sprintf("drv-%d", idx)
		vehicleID := fmt.Sprintf("veh-%d", idx)
		driverName := fmt.Sprintf("Driver %d", idx)
		vehicleName := fmt.Sprintf("Truck %d", idx)
		fixture.Drivers = append(fixture.Drivers, Record{
			"id":   driverID,
			"name": driverName,
		})
		fixture.Assets = append(fixture.Assets, Record{
			"id":           vehicleID,
			"name":         vehicleName,
			"type":         "vehicle",
			"licensePlate": fmt.Sprintf("TX-%04d", idx),
			"vin":          fmt.Sprintf("VIN%04dSIM", idx),
		})
		fixture.Routes = append(fixture.Routes, Record{
			"id":      fmt.Sprintf("route-%d", idx),
			"name":    fmt.Sprintf("Route %d", idx),
			"driver":  map[string]any{"id": driverID, "name": driverName},
			"vehicle": map[string]any{"id": vehicleID, "name": vehicleName},
		})
	}
	return fixture
}

func newDvirTestServer(t *testing.T, driverCount int) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Webhooks.Enabled = false

	store := NewStore(newDvirFixture(driverCount))
	scenarios, err := NewScenarioEngine("dvir-test-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	return NewServer(&cfg, store, scenarios, nil, nil)
}

func TestServerDvirHistoryParamValidation(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	now := srv.simNow()
	startRaw := url.QueryEscape(now.Add(-24 * time.Hour).Format(time.RFC3339))
	endRaw := url.QueryEscape(now.Format(time.RFC3339))

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing time range",
			target:     "/fleet/dvirs/history",
			wantStatus: http.StatusBadRequest,
			wantCode:   "TIME_RANGE_REQUIRED",
		},
		{
			name:       "missing end time",
			target:     "/fleet/dvirs/history?startTime=" + startRaw,
			wantStatus: http.StatusBadRequest,
			wantCode:   "TIME_RANGE_REQUIRED",
		},
		{
			name:       "malformed start time",
			target:     "/fleet/dvirs/history?startTime=not-a-time&endTime=" + endRaw,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "end before start",
			target:     "/fleet/dvirs/history?startTime=" + endRaw + "&endTime=" + startRaw,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name: "range wider than 30 days",
			target: "/fleet/dvirs/history?startTime=" + url.QueryEscape(
				now.Add(-40*24*time.Hour).Format(time.RFC3339),
			) + "&endTime=" + endRaw,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
		},
		{
			name:       "valid range",
			target:     "/fleet/dvirs/history?startTime=" + startRaw + "&endTime=" + endRaw,
			wantStatus: http.StatusOK,
		},
	}

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

func TestLiveSimulatorDvirsDeterministic(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(NewStore(newDvirFixture(3)), "dvir-deterministic-seed")
	now := simulator.anchorTime.Add(72 * time.Hour)
	windowStart := now.Add(-48 * time.Hour)

	first := simulator.Dvirs(now, windowStart, now, nil, nil)
	second := simulator.Dvirs(now, windowStart, now, nil, nil)
	if len(first) == 0 {
		t.Fatal("expected generated DVIR records")
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("expected identical DVIR output for identical sim time")
	}
}

func TestLiveSimulatorDvirsPreAndPostTripPerDay(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(NewStore(newDvirFixture(3)), "dvir-day-seed")
	now := simulator.anchorTime.Add(72 * time.Hour)
	day := now.Add(-48 * time.Hour).Truncate(24 * time.Hour)

	records := simulator.Dvirs(now, day, day.Add(26*time.Hour), []string{"drv-1"}, nil)
	if len(records) != 2 {
		t.Fatalf("expected pre-trip and post-trip DVIRs for one day, got %d", len(records))
	}

	byType := map[string]Record{}
	for _, record := range records {
		byType[stringValue(record, "type")] = record
		if nestedString(record, "driver", "id") != "drv-1" {
			t.Fatalf("unexpected driver id %q", nestedString(record, "driver", "id"))
		}
		if nestedString(record, "vehicle", "id") != "veh-1" {
			t.Fatalf("unexpected vehicle id %q", nestedString(record, "vehicle", "id"))
		}
		if stringValue(record, "licensePlate") == "" {
			t.Fatal("expected licensePlate on DVIR record")
		}
		if stringValue(record, "location") == "" {
			t.Fatal("expected location string on DVIR record")
		}
		if int64(floatFromAny(record["odometerMeters"])) <= 0 {
			t.Fatalf("expected positive odometerMeters, got %v", record["odometerMeters"])
		}

		startTime := mustParseRecordTime(t, record, "startTime")
		endTime := mustParseRecordTime(t, record, "endTime")
		if !endTime.After(startTime) {
			t.Fatalf("expected endTime after startTime, got %s / %s", startTime, endTime)
		}

		signature, ok := anyAsMap(record["authorSignature"])
		if !ok {
			t.Fatal("expected authorSignature object")
		}
		if stringValue(Record(signature), "type") != dvirSignatureTypeDriver {
			t.Fatalf("unexpected signature type %q", stringValue(Record(signature), "type"))
		}
		if stringValue(Record(signature), "signedAtTime") != stringValue(record, "endTime") {
			t.Fatal("expected signedAtTime to match endTime")
		}
		if nestedString(record, "authorSignature", "signatoryUser", "id") != "drv-1" {
			t.Fatal("expected signatoryUser to match driver")
		}
	}

	preTrip, hasPre := byType[dvirTypePreTrip]
	postTrip, hasPost := byType[dvirTypePostTrip]
	if !hasPre || !hasPost {
		t.Fatalf("expected one preTrip and one postTrip DVIR, got %v", byType)
	}
	if stringValue(preTrip, "endTime") > stringValue(postTrip, "startTime") {
		t.Fatal("expected pre-trip DVIR to complete before post-trip DVIR begins")
	}
}

func TestLiveSimulatorDvirsUnsafeRateAndDefects(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(NewStore(newDvirFixture(40)), "dvir-rate-seed")
	now := simulator.anchorTime.Add(9 * 24 * time.Hour)
	windowStart := now.Add(-6 * 24 * time.Hour)

	records := simulator.Dvirs(now, windowStart, now, nil, nil)
	if len(records) < 200 {
		t.Fatalf("expected large DVIR sample across roster, got %d", len(records))
	}

	flagged := 0
	sawResolved := false
	sawUnresolved := false
	catalogTypes := map[string]struct{}{}
	for _, entry := range dvirDefectCatalog {
		catalogTypes[entry.DefectType] = struct{}{}
	}

	for _, record := range records {
		status := stringValue(record, "safetyStatus")
		endTime := mustParseRecordTime(t, record, "endTime")
		defects, hasDefects := record["vehicleDefects"].([]any)

		if status == dvirSafetyStatusSafe {
			if hasDefects {
				t.Fatal("expected no vehicleDefects on safe DVIR")
			}
			continue
		}
		flagged++
		if !hasDefects || len(defects) < 1 || len(defects) > 2 {
			t.Fatalf("expected 1-2 defects on %q DVIR, got %d", status, len(defects))
		}

		expectResolved := now.Sub(endTime) >= dvirDefectResolveAge
		if expectResolved && status != dvirSafetyStatusResolved {
			t.Fatalf("expected resolved safetyStatus for old DVIR, got %q", status)
		}
		if !expectResolved && status != dvirSafetyStatusUnsafe {
			t.Fatalf("expected unsafe safetyStatus for recent DVIR, got %q", status)
		}

		for _, rawDefect := range defects {
			defect, ok := anyAsMap(rawDefect)
			if !ok {
				t.Fatal("expected defect object")
			}
			if _, known := catalogTypes[stringValue(Record(defect), "defectType")]; !known {
				t.Fatalf("unexpected defectType %q", stringValue(Record(defect), "defectType"))
			}
			if stringValue(Record(defect), "comment") == "" {
				t.Fatal("expected defect comment")
			}
			isResolved, isBool := defect["isResolved"].(bool)
			if !isBool || isResolved != expectResolved {
				t.Fatalf("expected isResolved=%v, got %v", expectResolved, defect["isResolved"])
			}
			resolvedAt := stringValue(Record(defect), "resolvedAtTime")
			if expectResolved {
				sawResolved = true
				if resolvedAt == "" || resolvedAt > now.UTC().Format(time.RFC3339) {
					t.Fatalf("expected resolvedAtTime in the past, got %q", resolvedAt)
				}
			} else {
				sawUnresolved = true
				if resolvedAt != "" {
					t.Fatalf("expected no resolvedAtTime on unresolved defect, got %q", resolvedAt)
				}
			}
		}
	}

	fraction := float64(flagged) / float64(len(records))
	if fraction < 0.02 || fraction > 0.25 {
		t.Fatalf("expected ~10%% unsafe DVIR rate, got %.3f (%d/%d)", fraction, flagged, len(records))
	}
	if !sawResolved || !sawUnresolved {
		t.Fatalf(
			"expected both resolved and unresolved defects, resolved=%v unresolved=%v",
			sawResolved,
			sawUnresolved,
		)
	}
}

func TestServerDvirHistoryFiltersAndPagination(t *testing.T) {
	t.Parallel()

	srv := newDvirTestServer(t, 3)
	now := srv.simNow()
	startRaw := url.QueryEscape(now.Add(-48 * time.Hour).Format(time.RFC3339))
	endRaw := url.QueryEscape(now.Format(time.RFC3339))
	base := "/fleet/dvirs/history?startTime=" + startRaw + "&endTime=" + endRaw

	byDriver := performAuthorizedRequest(srv, http.MethodGet, base+"&driverIds=drv-1")
	if byDriver.Code != http.StatusOK {
		t.Fatalf("expected 200 for driver filter, got %d", byDriver.Code)
	}
	driverRecords := mustReadDataRecords(t, byDriver.Body.Bytes())
	if len(driverRecords) == 0 {
		t.Fatal("expected DVIRs for filtered driver")
	}
	for _, record := range driverRecords {
		if nestedString(record, "driver", "id") != "drv-1" {
			t.Fatalf("unexpected driver id %q", nestedString(record, "driver", "id"))
		}
	}

	byVehicle := performAuthorizedRequest(srv, http.MethodGet, base+"&vehicleIds=veh-2")
	if byVehicle.Code != http.StatusOK {
		t.Fatalf("expected 200 for vehicle filter, got %d", byVehicle.Code)
	}
	vehicleRecords := mustReadDataRecords(t, byVehicle.Body.Bytes())
	if len(vehicleRecords) == 0 {
		t.Fatal("expected DVIRs for filtered vehicle")
	}
	for _, record := range vehicleRecords {
		if nestedString(record, "vehicle", "id") != "veh-2" {
			t.Fatalf("unexpected vehicle id %q", nestedString(record, "vehicle", "id"))
		}
	}

	firstPage := performAuthorizedRequest(srv, http.MethodGet, base+"&limit=2")
	if firstPage.Code != http.StatusOK {
		t.Fatalf("expected 200 for first page, got %d", firstPage.Code)
	}
	firstRecords, firstPagination := mustReadDailyLogPage(t, firstPage.Body.Bytes())
	if len(firstRecords) != 2 {
		t.Fatalf("expected 2 records on first page, got %d", len(firstRecords))
	}
	if hasNext, ok := firstPagination["hasNextPage"].(bool); !ok || !hasNext {
		t.Fatalf("expected hasNextPage true, got %v", firstPagination["hasNextPage"])
	}
	endCursor, ok := firstPagination["endCursor"].(string)
	if !ok || endCursor == "" {
		t.Fatalf("expected endCursor, got %v", firstPagination["endCursor"])
	}

	secondPage := performAuthorizedRequest(
		srv,
		http.MethodGet,
		base+"&limit=2&after="+url.QueryEscape(endCursor),
	)
	if secondPage.Code != http.StatusOK {
		t.Fatalf("expected 200 for second page, got %d", secondPage.Code)
	}
	secondRecords, _ := mustReadDailyLogPage(t, secondPage.Body.Bytes())
	if len(secondRecords) == 0 {
		t.Fatal("expected records on second page")
	}
	if stringValue(secondRecords[0], "id") == stringValue(firstRecords[0], "id") ||
		stringValue(secondRecords[0], "id") == stringValue(firstRecords[1], "id") {
		t.Fatal("expected second page to continue past the first page")
	}

	invalidCursor := performAuthorizedRequest(
		srv,
		http.MethodGet,
		base+"&limit=2&after=missing-cursor",
	)
	if invalidCursor.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid cursor, got %d", invalidCursor.Code)
	}
	if code := mustReadErrorCode(t, invalidCursor.Body.Bytes()); code != "INVALID_CURSOR" {
		t.Fatalf("expected INVALID_CURSOR code, got %q", code)
	}
}

type webhookEventCapture struct {
	mu     sync.Mutex
	events []WebhookEvent
}

func (c *webhookEventCapture) handler(t *testing.T) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		event := WebhookEvent{}
		if err := sonic.ConfigDefault.NewDecoder(request.Body).Decode(&event); err != nil {
			t.Errorf("decode webhook payload: %v", err)
		}
		c.mu.Lock()
		c.events = append(c.events, event)
		c.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (c *webhookEventCapture) snapshot() []WebhookEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]WebhookEvent{}, c.events...)
}

func TestServerDispatchDvirWebhooksDeduplicated(t *testing.T) {
	capture := &webhookEventCapture{}
	webhookReceiver := httptest.NewServer(capture.handler(t))
	defer webhookReceiver.Close()

	srv := newEventTestServer(t, webhookReceiver.URL)
	now := srv.simNow()
	dvirs := srv.live.Dvirs(now, now.Add(-72*time.Hour), now, nil, nil)
	if len(dvirs) == 0 {
		t.Fatal("expected generated DVIRs for webhook test")
	}

	target := dvirs[len(dvirs)-1]
	targetEnd := mustParseRecordTime(t, Record(target), "endTime")
	at := targetEnd.Add(time.Minute)
	windowStart := at.Add(-defaultAssetLookback)
	expected := 0
	for _, record := range dvirs {
		endTime := mustParseRecordTime(t, Record(record), "endTime")
		if endTime.After(windowStart) && !endTime.After(at) {
			expected++
		}
	}
	if expected == 0 {
		t.Fatal("expected at least one DVIR inside the lazy dispatch window")
	}

	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats?vehicleIds=veh-1", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	srv.dispatchDvirEvents(request, at)
	srv.dispatchDvirEvents(request, at)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(capture.snapshot()) >= expected {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)

	events := capture.snapshot()
	if len(events) != expected {
		t.Fatalf("expected %d deduplicated DvirSubmitted deliveries, got %d", expected, len(events))
	}
	sawTarget := false
	for _, event := range events {
		if event.EventType != "DvirSubmitted" {
			t.Fatalf("unexpected event type %q", event.EventType)
		}
		data, ok := anyAsMap(event.Data)
		if !ok {
			t.Fatal("expected webhook data object")
		}
		dvir, ok := anyAsMap(data["dvir"])
		if !ok {
			t.Fatal("expected dvir payload in webhook data")
		}
		if _, hasDriver := anyAsMap(data["driver"]); !hasDriver {
			t.Fatal("expected driver payload in webhook data")
		}
		if _, hasVehicle := anyAsMap(data["vehicle"]); !hasVehicle {
			t.Fatal("expected vehicle payload in webhook data")
		}
		if stringValue(Record(dvir), "id") == recordID(target) {
			sawTarget = true
		}
	}
	if !sawTarget {
		t.Fatalf("expected DvirSubmitted delivery for %q", recordID(target))
	}
}
