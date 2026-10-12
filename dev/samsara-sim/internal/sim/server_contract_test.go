package sim

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

var requestIDPattern = regexp.MustCompile(`^[0-9a-f]{8}$`)

func newDefaultFixtureServer(t *testing.T, configure func(cfg *config.Config)) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.RateLimits.Enabled = false
	cfg.Webhooks.Enabled = false
	cfg.Simulation.ScriptPath = ""
	if configure != nil {
		configure(&cfg)
	}
	scenarios, err := NewScenarioEngine("contract-seed", "default")
	if err != nil {
		t.Fatalf("initialize scenario engine: %v", err)
	}
	return NewServer(&cfg, loadDefaultFixtureStore(t), scenarios, nil, nil)
}

func performRequestWithToken(
	srv *Server,
	method string,
	target string,
	token string,
	body any,
) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		encoded, err := sonic.Marshal(body)
		if err != nil {
			panic(err)
		}
		reader = bytes.NewReader(encoded)
	}
	var request *http.Request
	if reader != nil {
		request = httptest.NewRequest(method, target, reader)
		request.Header.Set("Content-Type", "application/json")
	} else {
		request = httptest.NewRequest(method, target, nil)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	srv.withMiddleware(srv.mux).ServeHTTP(response, request)
	return response
}

func mustReadAPIError(t *testing.T, body []byte) string {
	t.Helper()

	payload := map[string]any{}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error payload: %v (%s)", err, body)
	}
	if len(payload) != 2 {
		t.Fatalf("expected error envelope with exactly message and requestId, got %v", payload)
	}
	message, ok := payload["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		t.Fatalf("expected non-empty string message, got %v", payload["message"])
	}
	requestID, ok := payload["requestId"].(string)
	if !ok || !requestIDPattern.MatchString(requestID) {
		t.Fatalf("expected 8-hex requestId, got %v", payload["requestId"])
	}
	return message
}

func assertAPIErrorMessage(t *testing.T, body []byte, want error) {
	t.Helper()

	message := mustReadAPIError(t, body)
	prefix := strings.TrimSuffix(apiErrorMessage(want), ".")
	if !strings.HasPrefix(message, prefix) {
		t.Fatalf("expected error message starting with %q, got %q", prefix, message)
	}
}

func TestAPIErrorMessageFormatsSentence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{err: ErrUnauthorized, want: "Invalid token."},
		{err: ErrRecordNotFound, want: "Object not found."},
		{err: ErrRateLimitExceeded, want: "Exceeded rate limit."},
		{
			err:  methodNotAllowedError{Method: http.MethodDelete, Path: "/endpoint"},
			want: "DELETE not allowed on /endpoint.",
		},
	}
	for _, testCase := range tests {
		if got := apiErrorMessage(testCase.err); got != testCase.want {
			t.Fatalf("expected %q, got %q", testCase.want, got)
		}
	}
}

func TestServerRejectsWriteOnReadOnlyToken(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performRequestWithToken(
		srv,
		http.MethodPost,
		"/addresses",
		"dev-samsara-token-readonly",
		map[string]any{"name": "Read Only Address"},
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for read-only token write, got %d", response.Code)
	}
	assertAPIErrorMessage(t, response.Body.Bytes(), ErrForbidden)

	readResponse := performRequestWithToken(
		srv,
		http.MethodGet,
		"/addresses",
		"dev-samsara-token-readonly",
		nil,
	)
	if readResponse.Code != http.StatusOK {
		t.Fatalf("expected read-only token to read, got %d", readResponse.Code)
	}
}

func TestServerAuthenticationErrorsUseEnvelope(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	tests := []struct {
		name   string
		header string
		want   error
	}{
		{name: "missing header", header: "", want: ErrInvalidAuthorization},
		{name: "wrong scheme", header: "Basic abc", want: ErrInvalidAuthorization},
		{name: "empty bearer", header: "Bearer   ", want: ErrInvalidAuthorization},
		{name: "unknown token", header: "Bearer not-a-real-token", want: ErrUnauthorized},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/fleet/drivers", nil)
			if testCase.header != "" {
				request.Header.Set("Authorization", testCase.header)
			}
			response := httptest.NewRecorder()
			srv.withMiddleware(srv.mux).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected JSON content type, got %q", got)
			}
			assertAPIErrorMessage(t, response.Body.Bytes(), testCase.want)
		})
	}

	unknownPath := performRequestWithToken(srv, http.MethodGet, "/no/such/path", "", nil)
	if unknownPath.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401 before routing for unauthenticated request, got %d",
			unknownPath.Code,
		)
	}
}

func TestServerUnknownPathReturnsJSON404(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	for _, target := range []string{"/fleet/nope", "/v1/fleet/unknown", "/_sim/does-not-exist"} {
		response := performAuthorizedRequest(srv, http.MethodGet, target)
		if response.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for %s, got %d", target, response.Code)
		}
		if got := response.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON 404 for %s, got content type %q", target, got)
		}
		message := mustReadAPIError(t, response.Body.Bytes())
		if !strings.Contains(message, target) {
			t.Fatalf("expected 404 message to name %s, got %q", target, message)
		}
	}
}

func TestServerWrongMethodReturnsJSON405WithAllow(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	tests := []struct {
		method string
		target string
		allow  []string
	}{
		{
			method: http.MethodPut,
			target: "/addresses",
			allow:  []string{http.MethodGet, http.MethodHead, http.MethodPost},
		},
		{
			method: http.MethodPost,
			target: "/addresses/41226316",
			allow: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPatch,
				http.MethodDelete,
			},
		},
		{
			method: http.MethodDelete,
			target: "/fleet/hos/clocks",
			allow:  []string{http.MethodGet, http.MethodHead},
		},
		{
			method: http.MethodPatch,
			target: "/_sim/time",
			allow:  []string{http.MethodGet, http.MethodHead, http.MethodPut},
		},
	}
	for _, testCase := range tests {
		response := performAuthorizedRequestWithBody(
			srv,
			testCase.method,
			testCase.target,
			map[string]any{},
		)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf(
				"expected 405 for %s %s, got %d",
				testCase.method,
				testCase.target,
				response.Code,
			)
		}
		if got := response.Header().Get("Allow"); got != strings.Join(testCase.allow, ", ") {
			t.Fatalf("expected Allow %v for %s, got %q", testCase.allow, testCase.target, got)
		}
		message := mustReadAPIError(t, response.Body.Bytes())
		want := testCase.method + " not allowed on " + testCase.target + "."
		if message != want {
			t.Fatalf("expected %q, got %q", want, message)
		}
	}
}

func TestServerPublicPathWrongMethodReturnsJSON405(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performRequestWithToken(srv, http.MethodPost, "/_sim/health", "", map[string]any{})
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /_sim/health, got %d", response.Code)
	}
	if got := response.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("expected Allow GET, HEAD, got %q", got)
	}
	mustReadAPIError(t, response.Body.Bytes())

	health := performRequestWithToken(srv, http.MethodGet, "/_sim/health", "", nil)
	if health.Code != http.StatusOK {
		t.Fatalf("expected public health without auth, got %d", health.Code)
	}
}

func TestServerStrictPaginationValidation(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	for _, target := range []string{
		"/addresses?limit=0",
		"/addresses?limit=513",
		"/addresses?limit=ten",
		"/live-shares?limit=101",
		"/fleet/hos/clocks?limit=-1",
	} {
		response := performAuthorizedRequest(srv, http.MethodGet, target)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", target, response.Code)
		}
		assertAPIErrorMessage(t, response.Body.Bytes(), ErrLimitInvalid)
	}

	for _, target := range []string{
		"/addresses?limit=512",
		"/live-shares?limit=100",
		"/assets?limit=0",
		"/fleet/hos/logs?limit=99999",
		"/addresses?unknownParam=1",
	} {
		response := performAuthorizedRequest(srv, http.MethodGet, target)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", target, response.Code)
		}
	}

	invalidCursor := performAuthorizedRequest(srv, http.MethodGet, "/assets?after=missing-cursor")
	if invalidCursor.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid cursor, got %d", invalidCursor.Code)
	}
	assertAPIErrorMessage(t, invalidCursor.Body.Bytes(), ErrCursorInvalid)
}

func TestServerHOSClocksHonorsLimitAndResumesCursor(t *testing.T) {
	t.Parallel()

	srv := newDefaultFixtureServer(t, nil)
	full := performAuthorizedRequest(srv, http.MethodGet, "/fleet/hos/clocks")
	if full.Code != http.StatusOK {
		t.Fatalf("expected 200 for clocks, got %d", full.Code)
	}
	allClocks, fullPagination := mustReadDailyLogPage(t, full.Body.Bytes())
	if len(allClocks) < 5 {
		t.Fatalf("expected the fixture fleet's clocks, got %d", len(allClocks))
	}
	if fullPagination["hasNextPage"] != false || fullPagination["endCursor"] != "" {
		t.Fatalf("expected a single complete page, got %v", fullPagination)
	}

	seen := map[string]struct{}{}
	after := ""
	pages := 0
	for {
		target := "/fleet/hos/clocks?limit=2"
		if after != "" {
			target += "&after=" + url.QueryEscape(after)
		}
		response := performAuthorizedRequest(srv, http.MethodGet, target)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", target, response.Code)
		}
		records, pagination := mustReadDailyLogPage(t, response.Body.Bytes())
		pages++
		if len(records) == 0 || len(records) > 2 {
			t.Fatalf("expected 1-2 clocks per page, got %d", len(records))
		}
		for _, record := range records {
			driverID := nestedString(Record(record), "driver", "id")
			if _, dup := seen[driverID]; dup {
				t.Fatalf("driver %s returned on two pages", driverID)
			}
			seen[driverID] = struct{}{}
		}
		hasNext, _ := pagination["hasNextPage"].(bool)
		endCursor, _ := pagination["endCursor"].(string)
		if !hasNext {
			if endCursor != "" {
				t.Fatalf("expected empty endCursor on the last page, got %q", endCursor)
			}
			break
		}
		if endCursor == "" {
			t.Fatal("expected a non-empty endCursor while hasNextPage is true")
		}
		after = endCursor
	}
	if len(seen) != len(allClocks) {
		t.Fatalf("expected %d drivers across pages, got %d", len(allClocks), len(seen))
	}
	if pages != (len(allClocks)+1)/2 {
		t.Fatalf("expected %d pages, got %d", (len(allClocks)+1)/2, pages)
	}
}

func TestServerAssetListUsesFixedPageSize(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	for idx := 0; idx < 305; idx++ {
		if _, err := srv.store.Create(ResourceAssets, Record{
			"name": "Bulk Trailer",
			"type": "trailer",
		}); err != nil {
			t.Fatalf("create asset: %v", err)
		}
	}

	first := performAuthorizedRequest(srv, http.MethodGet, "/assets?limit=5")
	firstRecords, firstPagination := mustReadDailyLogPage(t, first.Body.Bytes())
	if len(firstRecords) != 300 {
		t.Fatalf("expected the fixed 300-asset page, got %d", len(firstRecords))
	}
	endCursor, _ := firstPagination["endCursor"].(string)
	if firstPagination["hasNextPage"] != true || endCursor == "" {
		t.Fatalf("expected another page, got %v", firstPagination)
	}

	second := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/assets?after="+url.QueryEscape(endCursor),
	)
	secondRecords, secondPagination := mustReadDailyLogPage(t, second.Body.Bytes())
	if len(secondRecords) != 306-300 {
		t.Fatalf("expected the remaining %d assets, got %d", 306-300, len(secondRecords))
	}
	if secondPagination["hasNextPage"] != false || secondPagination["endCursor"] != "" {
		t.Fatalf("expected the final page, got %v", secondPagination)
	}
}

func TestServerAssetListFiltersByType(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	if _, err := srv.store.Create(ResourceAssets, Record{
		"id":   "281474979348655",
		"name": "Trailer 1",
		"type": "trailer",
	}); err != nil {
		t.Fatalf("create trailer asset: %v", err)
	}
	for _, assetType := range []string{"vehicle", "trailer"} {
		response := performAuthorizedRequest(srv, http.MethodGet, "/assets?type="+assetType)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200 for type=%s, got %d", assetType, response.Code)
		}
		records := mustReadDataRecords(t, response.Body.Bytes())
		if len(records) == 0 {
			t.Fatalf("expected %s assets in fixture", assetType)
		}
		for _, record := range records {
			if got := stringValue(record, "type"); got != assetType {
				t.Fatalf("expected only %s assets, got %q (%s)", assetType, got, recordID(record))
			}
		}
	}

	invalid := performAuthorizedRequest(srv, http.MethodGet, "/assets?type=truck")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid asset type, got %d", invalid.Code)
	}
	assertAPIErrorMessage(t, invalid.Body.Bytes(), ErrAssetTypeInvalid)
}

func TestServerCreateIgnoresClientSuppliedID(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequestWithBody(srv, http.MethodPost, "/assets", map[string]any{
		"id":   testVehicleID,
		"name": "Truck 1001 duplicate",
		"type": "vehicle",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 for create, got %d", response.Code)
	}
	created := mustReadDataMap(t, response.Body.Bytes())
	createdID := stringValue(Record(created), "id")
	if createdID == testVehicleID {
		t.Fatal("expected the server to ignore the client-supplied id")
	}
	if mustNumericID(t, "asset", createdID) <= mustNumericID(t, "asset", testVehicleID) {
		t.Fatalf("expected a server-assigned id above existing ids, got %s", createdID)
	}

	original, err := srv.store.Get(ResourceAssets, testVehicleID)
	if err != nil {
		t.Fatalf("expected the existing asset to remain: %v", err)
	}
	if stringValue(original, "name") != "Truck 1001" {
		t.Fatalf(
			"expected the existing asset to be untouched, got %q",
			stringValue(original, "name"),
		)
	}

	for _, target := range []string{"/addresses", "/fleet/drivers", "/fleet/routes", "/webhooks"} {
		response = performAuthorizedRequestWithBody(srv, http.MethodPost, target, map[string]any{
			"id":               "client-chosen-id",
			"name":             "Client Named",
			"url":              "http://127.0.0.1:1/hook",
			"username":         "client.named",
			"password":         "Sup3rSecret!",
			"formattedAddress": "1 Main St, Austin, TX 78701",
			"geofence":         map[string]any{"circle": map[string]any{"radiusMeters": 100}},
		})
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200 for POST %s, got %d", target, response.Code)
		}
		payload := mustReadJSONMap(t, response.Body.Bytes())
		record := Record(payload)
		if data, ok := anyAsMap(payload["data"]); ok {
			record = Record(data)
		}
		id := recordID(record)
		if id == "client-chosen-id" {
			t.Fatalf("expected POST %s to assign its own id", target)
		}
		mustNumericID(t, target, id)
	}
}

func TestServerCreateAndPatchManageTimestamps(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	createdAt := time.Date(2026, time.March, 4, 15, 30, 0, 0, time.UTC)
	srv.clock.SetPaused(true)
	srv.clock.SetTime(createdAt)

	driverResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/fleet/drivers",
		map[string]any{
			"name":          "Timestamped Driver",
			"username":      "timestamped.driver",
			"password":      "Sup3rSecret!",
			"createdAtTime": "2001-01-01T00:00:00Z",
			"updatedAtTime": "2001-01-01T00:00:00Z",
		},
	)
	driver := Record(mustReadDataMap(t, driverResponse.Body.Bytes()))
	wantCreated := createdAt.Format(time.RFC3339)
	if stringValue(driver, "createdAtTime") != wantCreated ||
		stringValue(driver, "updatedAtTime") != wantCreated {
		t.Fatalf("expected server timestamps %s, got %v / %v",
			wantCreated, driver["createdAtTime"], driver["updatedAtTime"])
	}

	addressResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/addresses",
		map[string]any{
			"name":             "Timestamped Yard",
			"formattedAddress": "1 Main St, Austin, TX 78701",
			"geofence":         map[string]any{"circle": map[string]any{"radiusMeters": 120}},
		},
	)
	address := Record(mustReadDataMap(t, addressResponse.Body.Bytes()))
	if stringValue(address, "createdAtTime") != wantCreated {
		t.Fatalf("expected address createdAtTime %s, got %v", wantCreated, address["createdAtTime"])
	}
	if _, has := address["updatedAtTime"]; has {
		t.Fatal("expected addresses to carry no updatedAtTime, matching the spec object")
	}

	routeResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/fleet/routes",
		map[string]any{"name": "Untimestamped Route"},
	)
	route := Record(mustReadDataMap(t, routeResponse.Body.Bytes()))
	if _, has := route["createdAtTime"]; has {
		t.Fatal("expected routes to carry no createdAtTime, matching the spec object")
	}

	assetResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/assets",
		map[string]any{"name": "Timestamped Trailer", "type": "trailer"},
	)
	asset := Record(mustReadDataMap(t, assetResponse.Body.Bytes()))
	assetID := recordID(asset)

	patchedAt := createdAt.Add(90 * time.Minute)
	srv.clock.SetTime(patchedAt)
	patchResponse := performAuthorizedRequestWithBody(
		srv,
		http.MethodPatch,
		"/assets?id="+assetID,
		map[string]any{
			"name":          "Renamed Trailer",
			"createdAtTime": "2001-01-01T00:00:00Z",
		},
	)
	patched := Record(mustReadDataMap(t, patchResponse.Body.Bytes()))
	if stringValue(patched, "name") != "Renamed Trailer" {
		t.Fatalf("expected patch to apply, got %v", patched["name"])
	}
	if stringValue(patched, "createdAtTime") != wantCreated {
		t.Fatalf("expected createdAtTime to stay %s, got %v", wantCreated, patched["createdAtTime"])
	}
	if stringValue(patched, "updatedAtTime") != patchedAt.Format(time.RFC3339) {
		t.Fatalf("expected updatedAtTime %s, got %v",
			patchedAt.Format(time.RFC3339), patched["updatedAtTime"])
	}

	addressPatch := performAuthorizedRequestWithBody(
		srv,
		http.MethodPatch,
		"/addresses/"+recordID(address),
		map[string]any{"notes": "gate code 1234"},
	)
	patchedAddress := Record(mustReadDataMap(t, addressPatch.Body.Bytes()))
	if _, has := patchedAddress["updatedAtTime"]; has {
		t.Fatal("expected address patch not to add updatedAtTime")
	}
	if stringValue(patchedAddress, "createdAtTime") != wantCreated {
		t.Fatalf("expected address createdAtTime to stay, got %v", patchedAddress["createdAtTime"])
	}
}

func TestServerWebhookCreateGeneratesSecretAtomically(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequestWithBody(srv, http.MethodPost, "/webhooks", map[string]any{
		"name": "Secretless",
		"url":  "http://127.0.0.1:1/hook",
	})
	created := Record(mustReadJSONMap(t, response.Body.Bytes()))
	if stringValue(created, "secretKey") != generatedWebhookSecret(recordID(created)) {
		t.Fatalf("expected generated secret, got %v", created["secretKey"])
	}
	stored, err := srv.store.Get(ResourceWebhooks, recordID(created))
	if err != nil {
		t.Fatalf("get webhook: %v", err)
	}
	if stringValue(stored, "secretKey") != stringValue(created, "secretKey") {
		t.Fatal("expected the stored webhook to carry the generated secret")
	}
}

func TestServerRouteListIncludesLifecycleStops(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/routes?ids=4129806431")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 for route list, got %d", response.Code)
	}

	records := mustReadDataRecords(t, response.Body.Bytes())
	if len(records) != 1 {
		t.Fatalf("expected one route record, got %d", len(records))
	}
	route := records[0]
	if stringValue(Record(route), "status") == "" {
		t.Fatal("expected dynamic route status in route list")
	}
	rawStops, ok := route["stops"].([]any)
	if !ok || len(rawStops) < 3 {
		t.Fatalf("expected >=3 lifecycle stops, got %d", len(rawStops))
	}
	firstStop, ok := rawStops[0].(map[string]any)
	if !ok {
		t.Fatal("expected stop object payload")
	}
	if stringValue(Record(firstStop), "etaTime") == "" {
		t.Fatal("expected stop etaTime in lifecycle payload")
	}
}
