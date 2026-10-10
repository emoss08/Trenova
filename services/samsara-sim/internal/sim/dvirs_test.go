package sim

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

const (
	fixtureMechanicMarcus = "524878"
	fixtureAdminDana      = "524871"
)

func newDvirFixture(driverCount int) *Fixture {
	fixture := &Fixture{}
	for idx := 1; idx <= driverCount; idx++ {
		driverID := strconv.Itoa(1_654_972 + idx)
		vehicleID := strconv.Itoa(281_474_976_710_656 + idx)
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
			"id":      strconv.Itoa(4_129_806_430 + idx),
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
	cfg.RateLimits.Enabled = false
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Webhooks.Enabled = false

	store := NewStore(newDvirFixture(driverCount))
	scenarios, err := NewScenarioEngine("dvir-test-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	return NewServer(&cfg, store, scenarios, nil, nil)
}

type webhookEventCapture struct {
	mu     sync.Mutex
	events []WebhookEvent
	url    string
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

func dvirHistoryWindow(start, end time.Time, extra map[string]string) string {
	params := map[string]string{
		"startTime": start.Format(time.RFC3339),
		"endTime":   end.Format(time.RFC3339),
	}
	for key, value := range extra {
		params[key] = value
	}
	return query("/fleet/dvirs/history", params)
}

func collectDvirStream(t *testing.T, srv *Server, params map[string]string) ([]Record, string) {
	t.Helper()

	out := []Record{}
	after := ""
	for page := 0; page < 50; page++ {
		current := mergeParams(params, map[string]string{})
		if after != "" {
			current["after"] = after
		}
		result := callAPI(t, srv, http.MethodGet, query("/dvirs/stream", current), nil).expect(t, http.StatusOK)
		out = append(out, result.list(t)...)
		pagination := result.pagination(t)
		after = stringOf(pagination["endCursor"])
		if pagination["hasNextPage"] != true {
			return out, after
		}
	}
	t.Fatal("DVIR stream did not finish paging")
	return nil, ""
}

func findUnsafeDvir(t *testing.T, srv *Server) Record {
	t.Helper()

	records, _ := collectDvirStream(t, srv, map[string]string{
		"startTime":    fleetTestTime.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
		"endTime":      fleetTestTime.Format(time.RFC3339),
		"safetyStatus": "unsafe",
	})
	for _, record := range records {
		if record["vehicle"] != nil && len(listOf(record["defectIds"])) > 0 {
			return record
		}
	}
	t.Fatal("expected an unsafe DVIR with defects in the fixture timeline")
	return nil
}

func TestDvirHistoryValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	start := fleetTestTime.Add(-48 * time.Hour)

	callAPI(t, srv, http.MethodGet, "/fleet/dvirs/history", nil).
		expectError(t, http.StatusBadRequest, "startTime and endTime")
	callAPI(t, srv, http.MethodGet, query("/fleet/dvirs/history", map[string]string{
		"startTime": start.Format(time.RFC3339),
	}), nil).expectError(t, http.StatusBadRequest, "startTime and endTime")
	callAPI(t, srv, http.MethodGet, query("/fleet/dvirs/history", map[string]string{
		"startTime": "not-a-time",
		"endTime":   fleetTestTime.Format(time.RFC3339),
	}), nil).expectError(t, http.StatusBadRequest, "Invalid value for parameter startTime")
	callAPI(t, srv, http.MethodGet, dvirHistoryWindow(fleetTestTime, start, nil), nil).
		expectError(t, http.StatusBadRequest, "endTime")
	callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, map[string]string{"limit": "513"}), nil).
		expectError(t, http.StatusBadRequest, "limit")
	callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, map[string]string{"after": "nope"}), nil).
		expectError(t, http.StatusBadRequest, "after")
	callAPI(t, srv, http.MethodGet, dvirHistoryWindow(fleetTestTime.Add(-45*24*time.Hour), fleetTestTime, nil), nil).
		expect(t, http.StatusOK)
}

func TestDvirHistoryShapeFiltersAndPaging(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	start := fleetTestTime.Add(-10 * 24 * time.Hour)

	all := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, nil), nil).
		expect(t, http.StatusOK).list(t)
	if len(all) < 12*2*8 {
		t.Fatalf("expected pre- and post-trip DVIRs for every driver, got %d", len(all))
	}
	previous := ""
	sawResolved := false
	for _, record := range all {
		end := stringValue(record, "endTime")
		if end < previous {
			t.Fatalf("expected DVIRs ordered by endTime, got %s after %s", end, previous)
		}
		previous = end
		for key := range record {
			if key == "driver" || strings.HasPrefix(key, "sim") {
				t.Fatalf("expected only spec Dvir fields, got %q", key)
			}
		}
		dvirType := stringValue(record, "type")
		if dvirType != dvirTypePreTrip && dvirType != dvirTypePostTrip {
			t.Fatalf("unexpected generated DVIR type %q", dvirType)
		}
		signature := Record(mapOf(record["authorSignature"]))
		if stringValue(signature, "type") != dvirSignatureTypeDriver ||
			stringValue(signature, "signedAtTime") != end {
			t.Fatalf("expected a driver signature at submission, got %v", signature)
		}
		if _, ok := mapOf(record["vehicle"])["ExternalIds"]; !ok {
			t.Fatalf("expected vehicle ExternalIds, got %v", record["vehicle"])
		}
		if trailer := mapOf(record["trailer"]); trailer != nil &&
			stringValue(record, "trailerName") != stringOf(trailer["name"]) {
			t.Fatalf("expected trailerName to match the trailer, got %v", record)
		}
		if stringValue(record, "safetyStatus") == dvirSafetyStatusResolved {
			sawResolved = true
			assertResolvedDvir(t, record)
		}
	}
	if !sawResolved {
		t.Fatal("expected resolved DVIRs within ten days of history")
	}

	byTractorTag := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, map[string]string{
		"tagIds": tagAustinTerminal,
	}), nil).expect(t, http.StatusOK).list(t)
	if len(byTractorTag) == 0 || len(byTractorTag) >= len(all) {
		t.Fatalf("expected the Austin tag to select a subset, got %d of %d", len(byTractorTag), len(all))
	}
	byTrailerTag := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, map[string]string{
		"tagIds": tagReeferTrailers,
	}), nil).expect(t, http.StatusOK).list(t)
	for _, record := range byTrailerTag {
		if record["trailer"] == nil {
			t.Fatal("expected trailer tag filter to select DVIRs with a trailer")
		}
	}
	byParent := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, map[string]string{
		"parentTagIds": tagTexasOperations,
	}), nil).expect(t, http.StatusOK).list(t)
	if len(byParent) != len(all) {
		t.Fatalf("expected the root tag to select every DVIR, got %d of %d", len(byParent), len(all))
	}

	collected := []Record{}
	after := ""
	for {
		params := map[string]string{"limit": "50"}
		if after != "" {
			params["after"] = after
		}
		page := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(start, fleetTestTime, params), nil).
			expect(t, http.StatusOK)
		collected = append(collected, page.list(t)...)
		after = stringOf(page.pagination(t)["endCursor"])
		if page.pagination(t)["hasNextPage"] != true {
			break
		}
	}
	if strings.Join(listIDs(collected), ",") != strings.Join(listIDs(all), ",") {
		t.Fatal("expected cursor paging to return the same DVIRs in the same order")
	}
}

func assertResolvedDvir(t *testing.T, record Record) {
	t.Helper()

	second := Record(mapOf(record["secondSignature"]))
	if stringValue(second, "type") != dvirSignatureTypeMechanic ||
		nestedString(second, "signatoryUser", "name") == "" {
		t.Fatalf("expected a mechanic second signature on a resolved DVIR, got %v", record)
	}
	if stringValue(record, "mechanicNotes") == "" {
		t.Fatalf("expected mechanic notes on a resolved DVIR, got %v", record)
	}
	defects := append(listOf(record["vehicleDefects"]), listOf(record["trailerDefects"])...)
	if len(defects) == 0 {
		t.Fatalf("expected defects on a resolved DVIR, got %v", record)
	}
	for _, raw := range defects {
		defect := Record(mapOf(raw))
		if defect["isResolved"] != true || stringValue(defect, "resolvedAtTime") == "" ||
			nestedString(defect, "resolvedBy", "type") != dvirSignatureTypeMechanic ||
			stringValue(defect, "mechanicNotes") == "" {
			t.Fatalf("expected a mechanic-resolved defect, got %v", defect)
		}
		if defect["vehicle"] == nil && defect["trailer"] == nil {
			t.Fatalf("expected the defect to name its vehicle or trailer, got %v", defect)
		}
		if stringValue(defect, "resolvedAtTime") > stringValue(second, "signedAtTime") {
			t.Fatalf("expected the second signature after every resolution, got %v", record)
		}
	}
	if third := Record(mapOf(record["thirdSignature"])); third != nil {
		if stringValue(third, "type") != dvirSignatureTypeDriver ||
			nestedString(third, "signatoryUser", "id") != nestedString(record, "authorSignature", "signatoryUser", "id") ||
			stringValue(third, "signedAtTime") <= stringValue(second, "signedAtTime") {
			t.Fatalf("expected the driver to acknowledge after the repair, got %v", record)
		}
	}
}

func TestDvirStreamFeedSemantics(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	start := fleetTestTime.Add(-5 * 24 * time.Hour)

	callAPI(t, srv, http.MethodGet, "/dvirs/stream", nil).
		expectError(t, http.StatusBadRequest, "startTime: is required")
	callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": fleetTestTime.Format(time.RFC3339),
		"endTime":   start.Format(time.RFC3339),
	}), nil).expectError(t, http.StatusBadRequest, "endTime")
	callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime":    start.Format(time.RFC3339),
		"safetyStatus": "unknown",
	}), nil).expectError(t, http.StatusBadRequest, "safetyStatus")
	callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": start.Format(time.RFC3339),
		"limit":     "201",
	}), nil).expectError(t, http.StatusBadRequest, "limit")
	callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": start.Format(time.RFC3339),
		"after":     "eyJ2IjoxLCJrIjoiaWQ6MSIsIm8iOjF9",
	}), nil).expectError(t, http.StatusBadRequest, "after")
	callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime":          start.Format(time.RFC3339),
		"includeExternalIds": "yes",
	}), nil).expectError(t, http.StatusBadRequest, "includeExternalIds")

	first := callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": fleetTestTime.Add(-10 * 24 * time.Hour).Format(time.RFC3339),
	}), nil).expect(t, http.StatusOK)
	if len(first.list(t)) != 200 || first.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a default 200-record page, got %d", len(first.list(t)))
	}

	feed, caughtUp := collectDvirStream(t, srv, map[string]string{
		"startTime": start.Format(time.RFC3339),
		"limit":     "37",
	})
	if caughtUp == "" {
		t.Fatal("expected an open-ended stream to keep a cursor when caught up")
	}
	previous := ""
	seen := map[string]struct{}{}
	for _, record := range feed {
		updated := stringValue(record, "updatedAtTime")
		if updated < previous || updated < start.Format(time.RFC3339) {
			t.Fatalf("expected ascending updatedAtTime inside the window, got %s after %s", updated, previous)
		}
		previous = updated
		if _, duplicate := seen[recordID(record)]; duplicate {
			t.Fatalf("expected each DVIR once, got %s twice", recordID(record))
		}
		seen[recordID(record)] = struct{}{}
		for _, key := range []string{"authorSignature", "dvirSubmissionBeginTime", "dvirSubmissionTime", "id", "type", "updatedAtTime"} {
			if _, ok := record[key]; !ok {
				t.Fatalf("expected required stream field %s, got %v", key, record)
			}
		}
		if _, ok := mapOf(record["vehicle"])["externalIds"]; ok {
			t.Fatal("expected externalIds only with includeExternalIds=true")
		}
	}

	srv.clock.SetTime(fleetTestTime.Add(6 * time.Hour))
	resumed := callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": start.Format(time.RFC3339),
		"after":     caughtUp,
	}), nil).expect(t, http.StatusOK)
	for _, record := range resumed.list(t) {
		if stringValue(record, "updatedAtTime") < previous {
			t.Fatalf("expected only newer changes after the caught-up cursor, got %v", record["updatedAtTime"])
		}
	}
	if len(resumed.list(t)) == 0 {
		t.Fatal("expected new DVIR changes six hours later")
	}
	srv.clock.SetTime(fleetTestTime)

	bounded := callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime": start.Format(time.RFC3339),
		"endTime":   fleetTestTime.Add(-4 * 24 * time.Hour).Format(time.RFC3339),
	}), nil).expect(t, http.StatusOK)
	if bounded.pagination(t)["hasNextPage"] != false || stringOf(bounded.pagination(t)["endCursor"]) != "" {
		t.Fatalf("expected a bounded stream's last page to end the cursor, got %v", bounded.pagination(t))
	}

	external := callAPI(t, srv, http.MethodGet, query("/dvirs/stream", map[string]string{
		"startTime":          start.Format(time.RFC3339),
		"includeExternalIds": "true",
		"safetyStatus":       "resolved,unsafe",
		"limit":              "20",
	}), nil).expect(t, http.StatusOK).list(t)
	for _, record := range external {
		status := stringValue(record, "safetyStatus")
		if status != dvirSafetyStatusResolved && status != dvirSafetyStatusUnsafe {
			t.Fatalf("expected the safetyStatus filter to apply, got %s", status)
		}
		if _, ok := mapOf(record["vehicle"])["externalIds"]; !ok {
			t.Fatalf("expected vehicle externalIds with includeExternalIds, got %v", record["vehicle"])
		}
		if nestedAny(record, "authorSignature", "signatoryUser", "externalIds") == nil {
			t.Fatalf("expected driver signatory externalIds, got %v", record["authorSignature"])
		}
	}
}

func TestDvirGetByID(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	history := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(fleetTestTime.Add(-30*24*time.Hour), fleetTestTime, map[string]string{
		"limit": "1",
	}), nil).expect(t, http.StatusOK).list(t)
	if len(history) != 1 {
		t.Fatal("expected a DVIR from 30 days ago")
	}
	id := recordID(history[0])
	fetched := callAPI(t, srv, http.MethodGet, "/dvirs/"+id+"?includeExternalIds=true", nil).expect(t, http.StatusOK)
	if fetched.Payload["data"] != nil || stringOf(fetched.Payload["id"]) != id {
		t.Fatalf("expected an unwrapped DVIR object, got %s", fetched.Body)
	}
	if stringOf(fetched.Payload["dvirSubmissionTime"]) != stringValue(history[0], "endTime") ||
		stringOf(fetched.Payload["dvirSubmissionBeginTime"]) != stringValue(history[0], "startTime") {
		t.Fatalf("expected stream timestamps to match history, got %s", fetched.Body)
	}
	if _, ok := mapOf(fetched.Payload["vehicle"])["externalIds"]; !ok {
		t.Fatalf("expected vehicle externalIds, got %s", fetched.Body)
	}
	callAPI(t, srv, http.MethodGet, "/dvirs/123", nil).expectError(t, http.StatusNotFound, "Object not found")
	callAPI(t, srv, http.MethodGet, "/dvirs/"+id+"?includeExternalIds=maybe", nil).
		expectError(t, http.StatusBadRequest, "includeExternalIds")
}

func TestDvirCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	valid := func() map[string]any {
		return map[string]any{
			"authorId":     fixtureMechanicMarcus,
			"type":         "mechanic",
			"safetyStatus": "safe",
			"vehicleId":    fixtureTruck1001,
		}
	}
	with := func(key string, value any) map[string]any {
		body := valid()
		if value == nil {
			delete(body, key)
		} else {
			body[key] = value
		}
		return body
	}
	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{name: "missing author", body: with("authorId", nil), fragment: "authorId is required"},
		{name: "unknown author", body: with("authorId", "999"), fragment: "user in the organization"},
		{name: "driver as author", body: with("authorId", fixtureDriverAlex), fragment: "user in the organization"},
		{name: "missing type", body: with("type", nil), fragment: "type is required"},
		{name: "driver type", body: with("type", "preTrip"), fragment: "`mechanic`"},
		{name: "missing safety status", body: with("safetyStatus", nil), fragment: "safetyStatus is required"},
		{name: "resolved safety status", body: with("safetyStatus", "resolved"), fragment: "`safe`, `unsafe`"},
		{name: "no asset", body: with("vehicleId", nil), fragment: "trailerId must be provided"},
		{name: "unknown vehicle", body: with("vehicleId", "1"), fragment: "no vehicle matches"},
		{name: "trailer as vehicle", body: with("vehicleId", fixtureTrailer2042), fragment: "no vehicle matches"},
		{name: "vehicle as trailer", body: with("trailerId", fixtureTruck1002), fragment: "no trailer matches"},
		{name: "long plate", body: with("licensePlate", "ABCDEFGHIJKLM"), fragment: "at most 12"},
		{name: "fractional odometer", body: with("odometerMeters", 12.5), fragment: "integer"},
		{name: "negative odometer", body: with("odometerMeters", float64(-1)), fragment: "between"},
		{name: "string odometer", body: with("odometerMeters", "100"), fragment: "integer"},
		{name: "unknown defect", body: with("resolvedDefectIds", []any{"42"}), fragment: "no DVIR defect matches"},
		{name: "defect list type", body: with("resolvedDefectIds", "42"), fragment: "array"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/fleet/dvirs", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
	if stored, err := srv.store.List(ResourceDvirs); err != nil || len(stored) != 0 {
		t.Fatalf("expected failed creates to store nothing, got %d (%v)", len(stored), err)
	}
}

func TestDvirCreateMechanicDvirAndWebhook(t *testing.T) {
	capture := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true, WebhookURL: capture.url})

	created := callAPI(t, srv, http.MethodPost, "/fleet/dvirs", map[string]any{
		"authorId":      fixtureMechanicMarcus,
		"type":          "mechanic",
		"safetyStatus":  "safe",
		"vehicleId":     fixtureTruck1001,
		"trailerId":     fixtureTrailer2041,
		"mechanicNotes": "PM-A service complete",
		"location":      "100 Fleet Ave, Austin, TX 78701",
		"ignored":       "field",
	}).expect(t, http.StatusOK).data(t)
	id := recordID(created)
	numeric, err := strconv.ParseInt(id, 10, 64)
	if err != nil || numeric <= apiDvirIDFloor {
		t.Fatalf("expected a server-assigned numeric DVIR ID, got %q", id)
	}
	if stringValue(created, "type") != dvirTypeMechanic || stringValue(created, "safetyStatus") != "safe" ||
		nestedString(created, "authorSignature", "type") != dvirSignatureTypeMechanic ||
		nestedString(created, "authorSignature", "signatoryUser", "id") != fixtureMechanicMarcus ||
		nestedString(created, "authorSignature", "signatoryUser", "name") != "Marcus Bell" ||
		stringValue(created, "licensePlate") != "TX-1001" ||
		stringValue(created, "trailerName") != "Trailer 2041" ||
		stringValue(created, "location") != "100 Fleet Ave, Austin, TX 78701" ||
		stringValue(created, "mechanicNotes") != "PM-A service complete" ||
		created["odometerMeters"] == nil || created["ignored"] != nil {
		t.Fatalf("unexpected created DVIR %v", created)
	}

	fetched := callAPI(t, srv, http.MethodGet, "/dvirs/"+id, nil).expect(t, http.StatusOK)
	if stringOf(fetched.Payload["updatedAtTime"]) != fleetTestTime.Format(time.RFC3339) {
		t.Fatalf("expected updatedAtTime at creation, got %s", fetched.Body)
	}
	stream, _ := collectDvirStream(t, srv, map[string]string{
		"startTime": fleetTestTime.Add(-time.Minute).Format(time.RFC3339),
	})
	if !strings.Contains(strings.Join(listIDs(stream), ","), id) {
		t.Fatal("expected the mechanic DVIR in the stream")
	}
	history := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(fleetTestTime.Add(-time.Hour), fleetTestTime, nil), nil).
		expect(t, http.StatusOK).list(t)
	if !strings.Contains(strings.Join(listIDs(history), ","), id) {
		t.Fatal("expected the mechanic DVIR in history")
	}

	events := waitForWebhookEvents(t, capture, "DvirSubmitted", 1)
	data := webhookData(t, events[0])
	if data["driver"] != nil {
		t.Fatalf("expected no driver on a mechanic DVIR webhook, got %v", data["driver"])
	}
	dvir := Record(mapOf(data["dvir"]))
	if recordID(dvir) != id || stringValue(dvir, "type") != dvirTypeMechanic ||
		nestedString(dvir, "authorSignature", "signatoryUser", "name") != "Marcus Bell" ||
		dvir["needsCorrection"] != false || nestedString(dvir, "trailer", "id") != fixtureTrailer2041 {
		t.Fatalf("unexpected DvirSubmitted payload %v", dvir)
	}
	vehicle := Record(mapOf(data["vehicle"]))
	if stringValue(vehicle, "vin") == "" || nestedString(vehicle, "gateway", "serial") == "" ||
		stringValue(vehicle, "assetType") != "vehicle" {
		t.Fatalf("expected VehicleWithGatewayTiny, got %v", vehicle)
	}
}

func TestDvirMechanicResolvesDefectsAndPatch(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	unsafe := findUnsafeDvir(t, srv)
	unsafeID := recordID(unsafe)
	vehicleID := nestedString(unsafe, "vehicle", "id")
	defectIDs := listOf(unsafe["defectIds"])

	otherVehicle := fixtureTruck1001
	if vehicleID == otherVehicle {
		otherVehicle = fixtureTruck1002
	}
	callAPI(t, srv, http.MethodPost, "/fleet/dvirs", map[string]any{
		"authorId":          fixtureMechanicMarcus,
		"type":              "mechanic",
		"safetyStatus":      "safe",
		"vehicleId":         otherVehicle,
		"resolvedDefectIds": defectIDs,
	}).expectError(t, http.StatusBadRequest, "different vehicle or trailer")

	trailerID := nestedString(unsafe, "trailer", "id")
	body := map[string]any{
		"authorId":          fixtureMechanicMarcus,
		"type":              "mechanic",
		"safetyStatus":      "safe",
		"vehicleId":         vehicleID,
		"resolvedDefectIds": defectIDs,
		"mechanicNotes":     "Repaired at the Austin shop",
	}
	if trailerID != "" {
		body["trailerId"] = trailerID
	}
	callAPI(t, srv, http.MethodPost, "/fleet/dvirs", body).expect(t, http.StatusOK)

	parent := callAPI(t, srv, http.MethodGet, "/dvirs/"+unsafeID, nil).expect(t, http.StatusOK)
	if stringOf(parent.Payload["safetyStatus"]) != dvirSafetyStatusResolved ||
		stringOf(parent.Payload["updatedAtTime"]) != fleetTestTime.Format(time.RFC3339) ||
		nestedString(Record(parent.Payload), "secondSignature", "signatoryUser", "id") != fixtureMechanicMarcus {
		t.Fatalf("expected the resolved parent DVIR, got %s", parent.Body)
	}
	callAPI(t, srv, http.MethodPost, "/fleet/dvirs", body).expectError(t, http.StatusBadRequest, "already resolved")
	callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+unsafeID, map[string]any{
		"authorId": fixtureMechanicMarcus, "isResolved": true,
	}).expectError(t, http.StatusBadRequest, "safetyStatus is resolved")

	history := callAPI(t, srv, http.MethodGet, dvirHistoryWindow(fleetTestTime.Add(-4*24*time.Hour), fleetTestTime, nil), nil).
		expect(t, http.StatusOK).list(t)
	for _, record := range history {
		if recordID(record) != unsafeID {
			continue
		}
		for _, raw := range append(listOf(record["vehicleDefects"]), listOf(record["trailerDefects"])...) {
			defect := Record(mapOf(raw))
			if nestedString(defect, "resolvedBy", "id") != fixtureMechanicMarcus ||
				stringValue(defect, "mechanicNotes") != "Repaired at the Austin shop" {
				t.Fatalf("expected the API mechanic to resolve the defect, got %v", defect)
			}
		}
	}
}

func TestDvirPatchResolvesDvirs(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	unsafe := findUnsafeDvir(t, srv)
	id := recordID(unsafe)
	submitted := stringValue(unsafe, "dvirSubmissionTime")

	cases := []struct {
		name     string
		target   string
		body     map[string]any
		status   int
		fragment string
	}{
		{name: "missing author", target: id, body: map[string]any{"isResolved": true}, status: http.StatusBadRequest, fragment: "authorId is required"},
		{name: "missing isResolved", target: id, body: map[string]any{"authorId": fixtureAdminDana}, status: http.StatusBadRequest, fragment: "isResolved is required"},
		{name: "isResolved false", target: id, body: map[string]any{"authorId": fixtureAdminDana, "isResolved": false}, status: http.StatusBadRequest, fragment: "must be true"},
		{name: "unknown user", target: id, body: map[string]any{"authorId": "1", "isResolved": true}, status: http.StatusBadRequest, fragment: "user in the organization"},
		{name: "future signature", target: id, body: map[string]any{"authorId": fixtureAdminDana, "isResolved": true, "signedAtTime": fleetTestTime.Add(time.Hour).Format(time.RFC3339)}, status: http.StatusBadRequest, fragment: "future"},
		{name: "signature before submission", target: id, body: map[string]any{"authorId": fixtureAdminDana, "isResolved": true, "signedAtTime": "2026-01-01T00:00:00Z"}, status: http.StatusBadRequest, fragment: "before the DVIR was submitted"},
		{name: "malformed signature", target: id, body: map[string]any{"authorId": fixtureAdminDana, "isResolved": true, "signedAtTime": "noon"}, status: http.StatusBadRequest, fragment: "RFC 3339"},
		{name: "unknown dvir", target: "12345", body: map[string]any{"authorId": fixtureAdminDana, "isResolved": true}, status: http.StatusNotFound, fragment: "Object not found"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+testCase.target, testCase.body).
				expectError(t, testCase.status, testCase.fragment)
		})
	}

	signedAt := mustParseRFC3339(t, submitted).Add(30 * time.Minute)
	if signedAt.After(fleetTestTime) {
		signedAt = fleetTestTime
	}
	resolved := callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+id, map[string]any{
		"authorId":      fixtureAdminDana,
		"isResolved":    true,
		"mechanicNotes": "Verified repair on site",
		"signedAtTime":  signedAt.Format(time.RFC3339),
	}).expect(t, http.StatusOK).data(t)
	if stringValue(resolved, "safetyStatus") != dvirSafetyStatusResolved ||
		stringValue(resolved, "mechanicNotes") != "Verified repair on site" ||
		nestedString(resolved, "secondSignature", "signatoryUser", "id") != fixtureAdminDana ||
		nestedString(resolved, "secondSignature", "signedAtTime") != signedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected resolved DVIR %v", resolved)
	}
	fetched := callAPI(t, srv, http.MethodGet, "/dvirs/"+id, nil).expect(t, http.StatusOK)
	if stringOf(fetched.Payload["updatedAtTime"]) != fleetTestTime.Format(time.RFC3339) {
		t.Fatalf("expected the resolution to bump updatedAtTime, got %s", fetched.Body)
	}
	callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+id, map[string]any{
		"authorId": fixtureAdminDana, "isResolved": true,
	}).expectError(t, http.StatusBadRequest, "safetyStatus is resolved")

	mechanic := callAPI(t, srv, http.MethodPost, "/fleet/dvirs", map[string]any{
		"authorId":     fixtureMechanicMarcus,
		"type":         "mechanic",
		"safetyStatus": "unsafe",
		"trailerId":    fixtureTrailer2053,
	}).expect(t, http.StatusOK).data(t)
	if mechanic["vehicle"] != nil || nestedString(mechanic, "trailer", "id") != fixtureTrailer2053 ||
		mechanic["trailerName"] != nil {
		t.Fatalf("expected a trailer-only mechanic DVIR, got %v", mechanic)
	}
	safeAgain := callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+recordID(mechanic), map[string]any{
		"authorId":   fixtureMechanicMarcus,
		"isResolved": true,
	}).expect(t, http.StatusOK).data(t)
	if stringValue(safeAgain, "safetyStatus") != dvirSafetyStatusResolved {
		t.Fatalf("expected the API DVIR to resolve, got %v", safeAgain)
	}
}

func TestDvirWritesPersistAndReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	enableTestPersistence(t, srv.store, path)
	unsafe := findUnsafeDvir(t, srv)

	created := callAPI(t, srv, http.MethodPost, "/fleet/dvirs", map[string]any{
		"authorId":     fixtureMechanicMarcus,
		"type":         "mechanic",
		"safetyStatus": "safe",
		"vehicleId":    fixtureTruck1005,
	}).expect(t, http.StatusOK).data(t)
	callAPI(t, srv, http.MethodPatch, "/fleet/dvirs/"+recordID(unsafe), map[string]any{
		"authorId":   fixtureMechanicMarcus,
		"isResolved": true,
	}).expect(t, http.StatusOK)
	waitForStateWrites(t, srv.store, 1)
	if err := srv.store.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	restored := loadDefaultFixtureStore(t)
	status := enableTestPersistence(t, restored, path)
	if !status.Restored {
		t.Fatal("expected the persisted state to be restored")
	}
	dvirs, _ := restored.List(ResourceDvirs)
	overlays, _ := restored.List(ResourceDvirResolutions)
	if len(dvirs) != 1 || recordID(dvirs[0]) != recordID(created) || len(overlays) != 1 {
		t.Fatalf("expected the API DVIR and the resolution to persist, got %d DVIRs and %d resolutions", len(dvirs), len(overlays))
	}

	callAPI(t, srv, http.MethodPost, "/_sim/state/reset", map[string]any{}).expect(t, http.StatusOK)
	callAPI(t, srv, http.MethodGet, "/dvirs/"+recordID(created), nil).expect(t, http.StatusNotFound)
	reverted := callAPI(t, srv, http.MethodGet, "/dvirs/"+recordID(unsafe), nil).expect(t, http.StatusOK)
	if stringOf(reverted.Payload["safetyStatus"]) != dvirSafetyStatusUnsafe {
		t.Fatalf("expected reset to drop the resolution, got %s", reverted.Body)
	}
	again := callAPI(t, srv, http.MethodPost, "/fleet/dvirs", map[string]any{
		"authorId":     fixtureMechanicMarcus,
		"type":         "mechanic",
		"safetyStatus": "safe",
		"vehicleId":    fixtureTruck1005,
	}).expect(t, http.StatusOK).data(t)
	if recordID(again) != recordID(created) {
		t.Fatalf("expected IDs to repeat after reset, got %s then %s", recordID(created), recordID(again))
	}
}

func TestServerDispatchDvirWebhooksDeduplicated(t *testing.T) {
	capture := &webhookEventCapture{}
	webhookReceiver := httptest.NewServer(capture.handler(t))
	defer webhookReceiver.Close()

	srv := newEventTestServer(t, webhookReceiver.URL)
	now := srv.simNow()
	ctx := srv.live.newDvirContext(now)
	cores := srv.live.dvirCores(ctx, now.Add(-72*time.Hour), now)
	if len(cores) == 0 {
		t.Fatal("expected generated DVIRs for webhook test")
	}
	sortDvirs(cores, fieldEndTime)
	target := cores[len(cores)-1]
	at := mustParseRecordTime(t, target, "endTime").Add(time.Minute)
	windowStart := at.Add(-defaultAssetLookback)
	expected := 0
	for _, record := range cores {
		endTime := mustParseRecordTime(t, record, "endTime")
		if endTime.After(windowStart) && !endTime.After(at) {
			expected++
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats?vehicleIds=281474976710657", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	srv.dispatchDvirEvents(request, at)
	srv.dispatchDvirEvents(request, at)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(capture.snapshot()) < expected {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)

	events := capture.snapshot()
	if len(events) != expected {
		t.Fatalf("expected %d deduplicated DvirSubmitted deliveries, got %d", expected, len(events))
	}
	sawTarget := false
	for _, event := range events {
		data, ok := anyAsMap(event.Data)
		if event.EventType != "DvirSubmitted" || !ok {
			t.Fatalf("unexpected event %q", event.EventType)
		}
		dvir := Record(mapOf(data["dvir"]))
		for _, key := range []string{"authorSignature", "endTime", "id", "needsCorrection", "safetyStatus", "startTime", "type"} {
			if _, present := dvir[key]; !present {
				t.Fatalf("expected required webhook dvir.%s, got %v", key, dvir)
			}
		}
		if nestedString(Record(data), "driver", "id") == "" || data["vehicle"] == nil {
			t.Fatalf("expected driver and vehicle in the webhook data, got %v", data)
		}
		for _, raw := range listOf(dvir["defects"]) {
			defect := Record(mapOf(raw))
			if stringValue(defect, "defectSeverity") == "" ||
				(defect["vehicle"] == nil && defect["trailer"] == nil) {
				t.Fatalf("expected webhook defects with severity and asset, got %v", defect)
			}
		}
		if recordID(dvir) == recordID(target) {
			sawTarget = true
		}
	}
	if !sawTarget {
		t.Fatalf("expected DvirSubmitted delivery for %q", recordID(target))
	}
}

func TestLiveSimulatorDvirsDeterministicPerDriverDay(t *testing.T) {
	t.Parallel()

	srv := newDvirTestServer(t, 3)
	now := time.Date(2026, time.March, 10, 23, 0, 0, 0, time.UTC)
	first := srv.live.dvirCores(srv.live.newDvirContext(now), now.Add(-5*24*time.Hour), now)
	second := srv.live.dvirCores(srv.live.newDvirContext(now), now.Add(-5*24*time.Hour), now)
	if len(first) < 3*2*5 || len(first) != len(second) {
		t.Fatalf("expected two DVIRs per driver-day, got %d and %d", len(first), len(second))
	}
	for idx := range first {
		if recordID(first[idx]) != recordID(second[idx]) ||
			stringValue(first[idx], "safetyStatus") != stringValue(second[idx], "safetyStatus") {
			t.Fatal("expected deterministic DVIR generation")
		}
	}
}
