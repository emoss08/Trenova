package sim

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

var tinyPNG = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n', 0, 0, 0, 13, 'I', 'H', 'D', 'R'}

func validDriverBody(username string) map[string]any {
	return map[string]any{
		"name":     "Pat Smith",
		"username": username,
		"password": "aSecurePassword1234",
	}
}

func TestDriverListDefaultsToActiveAndFilters(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	all := requireStatus(t, srv, http.MethodGet, "/fleet/drivers", nil, http.StatusOK).list(t)
	if len(all) != 12 {
		t.Fatalf("expected the 12 active fixture drivers, got %d", len(all))
	}
	alex := requireRecord(t, all, fixtureDriverAlex)
	for _, field := range []string{
		"username", "createdAtTime", "updatedAtTime", "timezone", "eldSettings", "licenseNumber",
		"licenseState", "phone", "tags", "vehicleGroupTag", "peerGroupTag", "trailerGroupTag",
		"currentIdCardCode", "tachographCardNumber", "attributes",
	} {
		if _, ok := alex[field]; !ok {
			t.Fatalf("expected %s on fixture driver, got %v", field, alex)
		}
	}
	if alex["isDeactivated"] != false || alex["driverActivationStatus"] != "active" {
		t.Fatalf(
			"expected an active driver, got %v / %v",
			alex["isDeactivated"],
			alex["driverActivationStatus"],
		)
	}
	if _, has := alex["password"]; has {
		t.Fatal("expected drivers never to expose a password")
	}

	requireStatus(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverJordan, map[string]any{
		"driverActivationStatus": "deactivated",
	}, http.StatusOK)
	active := requireStatus(t, srv, http.MethodGet, "/fleet/drivers", nil, http.StatusOK).list(t)
	if _, has := idSet(active)[fixtureDriverJordan]; has || len(active) != 11 {
		t.Fatalf(
			"expected the default listing to omit deactivated drivers, got %v",
			listIDs(active),
		)
	}
	deactivated := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers?driverActivationStatus=deactivated",
		nil,
		http.StatusOK,
	).list(t)
	if !slices.Equal(listIDs(deactivated), []string{fixtureDriverJordan}) {
		t.Fatalf("expected only the deactivated driver, got %v", listIDs(deactivated))
	}
	callAPI(t, srv, http.MethodGet, "/fleet/drivers?driverActivationStatus=retired", nil).
		expectError(t, http.StatusBadRequest, "driverActivationStatus")

	hazmatAttribute := ""
	for _, raw := range alex["attributes"].([]any) {
		attribute := Record(raw.(map[string]any))
		if stringValue(attribute, "name") == "Hazmat Endorsement" {
			hazmatAttribute = stringValue(attribute, "id")
		}
	}
	byValueID := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers?attributeValueIds="+attributeValueID(
			hazmatAttribute,
			"Yes",
		),
		nil,
		http.StatusOK,
	).list(t)
	byName := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/fleet/drivers",
			map[string]string{"attributes": "Hazmat Endorsement:Yes"},
		),
		nil,
		http.StatusOK,
	).list(t)
	if len(byValueID) != 4 || !slices.Equal(listIDs(byValueID), listIDs(byName)) {
		t.Fatalf("expected the four hazmat drivers by value id and name, got %v and %v",
			listIDs(byValueID), listIDs(byName))
	}
	experienced := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/fleet/drivers",
			map[string]string{"attributes": "Years of Experience:range(20,)"},
		),
		nil,
		http.StatusOK,
	).list(t)
	for _, driver := range experienced {
		years := 0.0
		for _, raw := range driver["attributes"].([]any) {
			attribute := Record(raw.(map[string]any))
			if stringValue(attribute, "name") == "Years of Experience" {
				years = attributeNumberValues(attribute)[0]
			}
		}
		if years < 20 {
			t.Fatalf(
				"expected only drivers with 20+ years, got %s with %v",
				recordID(driver),
				years,
			)
		}
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		query(
			"/fleet/drivers",
			map[string]string{"attributes": "Medical Card Expiration:range(2027-01-01,)"},
		),
		nil,
	).
		expectError(t, http.StatusBadRequest, "numbers")
	callAPI(
		t,
		srv,
		http.MethodGet,
		query("/fleet/drivers", map[string]string{"attributes": "broken"}),
		nil,
	).
		expectError(t, http.StatusBadRequest, "attributes")

	created := requireStatus(t, srv, http.MethodGet,
		"/fleet/drivers?createdAfterTime=2026-01-12T00:00:00Z", nil, http.StatusOK).list(t)
	if len(created) != 2 {
		t.Fatalf("expected the two drivers created after Jan 12, got %v", listIDs(created))
	}
	updated := requireStatus(t, srv, http.MethodGet,
		"/fleet/drivers?updatedAfterTime=2026-03-04T14:00:00.000-01:00", nil, http.StatusOK).list(t)
	if len(updated) != 0 {
		t.Fatalf("expected no driver updated after the probe time, got %v", listIDs(updated))
	}
	callAPI(t, srv, http.MethodGet, "/fleet/drivers?updatedAfterTime=yesterday", nil).
		expectError(t, http.StatusBadRequest, "updatedAfterTime")

	first := requireStatus(t, srv, http.MethodGet, "/fleet/drivers?limit=5", nil, http.StatusOK)
	cursor := stringValue(first.pagination(t), "endCursor")
	second := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers?limit=5&after="+cursor,
		nil,
		http.StatusOK,
	).list(t)
	if len(first.list(t)) != 5 || len(second) != 5 ||
		listIDs(second)[0] == listIDs(first.list(t))[0] {
		t.Fatalf(
			"expected limit/after paging, got %v then %v",
			listIDs(first.list(t)),
			listIDs(second),
		)
	}
}

func TestDriverCreateValidation(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	with := func(key string, value any) map[string]any {
		body := validDriverBody("pat.smith")
		if value == nil {
			delete(body, key)
		} else {
			body[key] = value
		}
		return body
	}
	tests := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{name: "name required", body: with("name", nil), fragment: "name is required"},
		{name: "username required", body: with("username", nil), fragment: "username is required"},
		{name: "password required", body: with("password", nil), fragment: "password is required"},
		{name: "blank name", body: with("name", "   "), fragment: "blank"},
		{name: "name too long", body: with("name", strings.Repeat("a", 256)), fragment: "255"},
		{name: "username space", body: with("username", "pat smith"), fragment: "spaces"},
		{name: "username at", body: with("username", "pat@smith"), fragment: "@"},
		{
			name:     "username too long",
			body:     with("username", strings.Repeat("u", 190)),
			fragment: "189",
		},
		{name: "username taken", body: with("username", "ARIVERA"), fragment: "already taken"},
		{
			name:     "external id taken",
			body:     with("externalIds", map[string]any{"workerId": "worker-1001"}),
			fragment: "already assigned",
		},
		{
			name:     "bad external key",
			body:     with("externalIds", map[string]any{"bad key": "x"}),
			fragment: "letters",
		},
		{
			name:     "external value type",
			body:     with("externalIds", map[string]any{"k": []any{"x"}}),
			fragment: "string",
		},
		{
			name:     "underscore external key",
			body:     with("externalIds", map[string]any{"payroll_id": "x"}),
			fragment: "letters and digits",
		},
		{
			name:     "external value symbol",
			body:     with("externalIds", map[string]any{"payroll": "a/b"}),
			fragment: "@ . _",
		},
		{name: "locale enum", body: with("locale", "zz"), fragment: "locale"},
		{name: "timezone", body: with("timezone", "Mars/Olympus"), fragment: "timezone"},
		{name: "email", body: with("email", "not-an-email"), fragment: "email"},
		{
			name:     "eld day start",
			body:     with("eldDayStartHour", float64(6)),
			fragment: "eldDayStartHour",
		},
		{name: "notes length", body: with("notes", strings.Repeat("n", 4097)), fragment: "4096"},
		{name: "license state", body: with("licenseState", "XX"), fragment: "licenseState"},
		{name: "unknown tag", body: with("tagIds", []any{"1"}), fragment: "tag"},
		{name: "unknown group tag", body: with("vehicleGroupTagId", "1"), fragment: "tag"},
		{
			name:     "unknown static vehicle",
			body:     with("staticAssignedVehicleId", "1"),
			fragment: "vehicle",
		},
		{name: "date of birth", body: with("dateOfBirth", "1990-13-01"), fragment: "dateOfBirth"},
		{
			name:     "ruleset override incomplete",
			body:     with("usDriverRulesetOverride", map[string]any{"cycle": "Texas (7/70)"}),
			fragment: "restart",
		},
		{
			name: "ruleset override enum",
			body: with(
				"usDriverRulesetOverride",
				map[string]any{
					"cycle":             "Mars",
					"restart":           "None",
					"restbreak":         "None",
					"usStateToOverride": "TX",
				},
			),
			fragment: "cycle",
		},
		{
			name: "duplicate license",
			body: func() map[string]any {
				body := validDriverBody("pat.license")
				body["licenseState"] = "TX"
				body["licenseNumber"] = "88960665"
				return body
			}(),
			fragment: "license",
		},
		{
			name:     "profile image",
			body:     with("profileImageBase64", "aGVsbG8="),
			fragment: "JPEG or PNG",
		},
		{name: "profile url", body: with("profileImageUrl", "ftp://x"), fragment: "http"},
	}
	for _, testCase := range tests {
		callAPI(t, srv, http.MethodPost, "/fleet/drivers", testCase.body).
			expectError(t, http.StatusBadRequest, testCase.fragment)
	}
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers",
		"not an object",
	).expect(t, http.StatusBadRequest)
}

func TestDriverCreateGetPatchAndWebhooks(t *testing.T) {
	t.Parallel()

	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{WebhookURL: sink.url})
	body := validDriverBody("pat.smith")
	body["externalIds"] = map[string]any{"trenovaWorkerId": "wrk_01HXYZ"}
	body["tagIds"] = []any{tagAustinTerminal}
	body["vehicleGroupTagId"] = tagAustinTerminal
	body["licenseState"] = "TX"
	body["timezone"] = "America/Chicago"
	body["usDriverRulesetOverride"] = map[string]any{
		"cycle":             "Texas (7/70)",
		"restart":           "34-hour Restart",
		"restbreak":         "None",
		"usStateToOverride": "TX",
	}
	body["profileImageBase64"] = base64.StdEncoding.EncodeToString(tinyPNG)
	body["attributes"] = []any{map[string]any{"name": "CDL Class", "stringValues": []any{"B"}}}
	created := requireStatus(t, srv, http.MethodPost, "/fleet/drivers", body, http.StatusOK).data(t)
	driverID := recordID(created)
	if mustNumericID(t, "driver", driverID) <= 1655297 {
		t.Fatalf("expected a server-assigned driver id, got %s", driverID)
	}
	if _, has := created["password"]; has {
		t.Fatal("expected the create response to omit the password")
	}
	if created["createdAtTime"] != fleetTestTime.Format(time.RFC3339) ||
		created["driverActivationStatus"] != "active" ||
		!strings.HasPrefix(stringValue(created, "profileImageUrl"), "https://") ||
		nestedString(created, "vehicleGroupTag", "id") != tagAustinTerminal {
		t.Fatalf("unexpected created driver %v", created)
	}
	rulesets := driverRulesets(t, created)
	if stringValue(Record(rulesets[0]), "cycle") != "TX 70 hour / 7 day" ||
		stringValue(Record(rulesets[0]), "shift") != "Texas Intrastate" {
		t.Fatalf("expected eldSettings to follow the ruleset override, got %v", rulesets)
	}
	attribute := Record(created["attributes"].([]any)[0].(map[string]any))
	if !uuidPattern.MatchString(stringValue(attribute, "id")) {
		t.Fatalf("expected a resolved attribute id, got %v", attribute)
	}

	byExternal := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/trenovaWorkerId:wrk_01HXYZ",
		nil,
		http.StatusOK,
	).data(t)
	if recordID(byExternal) != driverID {
		t.Fatalf("expected the external ID to resolve the driver, got %s", recordID(byExternal))
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/trenovaWorkerId:missing",
		nil,
	).expectError(t, http.StatusNotFound, "driver")

	srv.clock.SetTime(fleetTestTime.Add(time.Hour))
	patched := requireStatus(t, srv, http.MethodPatch, "/fleet/drivers/"+driverID, map[string]any{
		"externalIds": map[string]any{
			"trenovaWorkerId": "wrk_01HXYZ",
			"payrollId":       "P-77",
		},
		"usDriverRulesetOverride": nil,
		"tagIds":                  []any{tagHazmatCertified},
		"password":                "rotated-password-1",
		"vehicleGroupTagId":       "",
	}, http.StatusOK).data(t)
	if nestedString(patched, fieldExternalIDs, "payrollId") != "P-77" ||
		patched["updatedAtTime"] != fleetTestTime.Add(time.Hour).Format(time.RFC3339) ||
		patched["createdAtTime"] != fleetTestTime.Format(time.RFC3339) {
		t.Fatalf("unexpected patched driver %v", patched)
	}
	if _, has := patched["usDriverRulesetOverride"]; has {
		t.Fatal("expected null to clear the ruleset override")
	}
	if _, has := patched["vehicleGroupTag"]; has {
		t.Fatal("expected a blank vehicleGroupTagId to clear the group tag")
	}
	if !slices.Equal(tagIDsOf(t, patched), []string{tagHazmatCertified}) {
		t.Fatalf("expected tagIds to replace membership, got %v", tagIDsOf(t, patched))
	}
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/fleet/drivers/"+driverID,
		map[string]any{"username": "arivera"},
	).
		expectError(t, http.StatusBadRequest, "already taken")
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+driverID, map[string]any{"name": nil}).
		expectError(t, http.StatusBadRequest, "null")
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/404404", map[string]any{"name": "x"}).
		expectError(t, http.StatusNotFound, "driver")

	createdEvents := waitForWebhookEvents(t, sink, eventDriverCreated, 1)
	updatedEvents := waitForWebhookEvents(t, sink, eventDriverUpdated, 1)
	for _, event := range append(createdEvents, updatedEvents...) {
		data := webhookData(t, event)
		driver, ok := anyAsMap(data["driver"])
		if !ok || stringValue(Record(driver), "id") != driverID {
			t.Fatalf("expected a {driver} payload, got %v", data)
		}
		if _, has := driver["password"]; has {
			t.Fatal("expected webhooks never to carry the password")
		}
	}
}

func TestDriverActivationSemantics(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"deactivatedAtTime": "2026-03-01T00:00:00Z",
	}).expectError(t, http.StatusBadRequest, "deactivated")
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"driverActivationStatus": "deactivated",
		"deactivatedAtTime":      fleetTestTime.Add(time.Hour).Format(time.RFC3339),
	}).expectError(t, http.StatusBadRequest, "future")
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"driverActivationStatus": "paused",
	}).expectError(t, http.StatusBadRequest, "driverActivationStatus")

	deactivatedAt := fleetTestTime.Add(-30 * time.Hour)
	driver := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/drivers/workerId:worker-1001",
		map[string]any{
			"driverActivationStatus": "deactivated",
			"deactivatedAtTime":      deactivatedAt.Format(time.RFC3339),
		},
		http.StatusOK,
	).data(t)
	if driver["isDeactivated"] != true || driver["driverActivationStatus"] != "deactivated" {
		t.Fatalf("expected a deactivated driver, got %v", driver)
	}
	view := srv.fleetView()
	intervals := view.assignmentIntervals(&assignmentQuery{
		WindowStart: &deactivatedAt,
		WindowEnd:   &fleetTestTime,
		DriverIDs:   map[string]struct{}{fixtureDriverAlex: {}},
		DriverApp:   true,
	})
	for _, interval := range intervals {
		if interval.End == nil || interval.End.After(deactivatedAt) {
			t.Fatalf("expected assignments to end by deactivation, got %+v", interval)
		}
	}
	if view.currentDriverOf(fixtureTruck1001) == fixtureDriverAlex {
		t.Fatal("expected a deactivated driver not to be the vehicle's current driver")
	}

	reactivated := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/drivers/"+fixtureDriverAlex,
		map[string]any{
			"driverActivationStatus": "active",
		},
		http.StatusOK,
	).data(t)
	if reactivated["isDeactivated"] != false {
		t.Fatalf("expected reactivation, got %v", reactivated)
	}
	stored, err := srv.store.Get(ResourceDrivers, fixtureDriverAlex)
	if err != nil {
		t.Fatalf("get driver: %v", err)
	}
	if _, has := stored[fieldSimDeactivatedAtTime]; has {
		t.Fatal("expected reactivation to clear the deactivation time")
	}
}

func TestRulesetOverrideCyclesMapToEldCycles(t *testing.T) {
	t.Parallel()

	eldCycles := []string{
		"USA 60 hour / 7 day", "USA 70 hour / 8 day", "AK 80 hour / 8 day", "AK 70 hour / 7 day",
		"CA 80 hour / 8 day", "CA 112 hour / 8 day", "FL 80 hour / 8 day", "FL 70 hour / 7 day",
		"NE 80 hour / 8 day", "NE 70 hour / 7 day", "NC 80 hour / 8 day", "NC 70 hour / 7 day",
		"OK 70 hour / 8 day", "OK 60 hour / 7 day", "OR 80 hour / 8 day", "OR 70 hour / 7 day",
		"SC 80 hour / 8 day", "SC 70 hour / 7 day", "TX 70 hour / 7 day", "WI 80 hour / 8 day",
		"WI 70 hour / 7 day",
	}
	for _, cycle := range usRulesetCycles {
		mapped, ok := eldCycleForOverride(cycle)
		if !ok || !slices.Contains(eldCycles, mapped) {
			t.Fatalf("expected %q to map onto the ELD cycle enum, got %q", cycle, mapped)
		}
	}
	if _, ok := eldCycleForOverride("Mars (1/2)"); ok {
		t.Fatal("expected unknown regions not to map")
	}
}
