package sim

import (
	"bytes"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"
)

const (
	testDriverID     = "1654973"
	testVehicleID    = "281474976710657"
	fixtureVehicleID = "281474977075805"
	fixtureRouteID   = "4129806431"
	fixtureUserID    = "524871"
)

var (
	uuidPattern = regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
	)
	liveShareIDPattern = regexp.MustCompile(`^[a-z0-9]{19}$`)
)

func mustNumericID(t *testing.T, label, id string) int64 {
	t.Helper()

	value, ok := numericID(id)
	if !ok {
		t.Fatalf("expected %s id to be a numeric string, got %q", label, id)
	}
	return value
}

func recordIDSet(records []Record) map[string]struct{} {
	out := make(map[string]struct{}, len(records))
	for _, record := range records {
		out[recordID(record)] = struct{}{}
	}
	return out
}

func TestDefaultFixtureUsesSamsaraIDFormats(t *testing.T) {
	t.Parallel()

	store := loadDefaultFixtureStore(t)
	list := func(resource Resource) []Record {
		records, err := store.List(resource)
		if err != nil {
			t.Fatalf("list %s: %v", resource, err)
		}
		return records
	}

	drivers := list(ResourceDrivers)
	assets := list(ResourceAssets)
	addresses := list(ResourceAddresses)
	routes := list(ResourceRoutes)
	webhooks := list(ResourceWebhooks)
	if len(drivers) == 0 || len(assets) == 0 || len(addresses) == 0 || len(routes) == 0 ||
		len(webhooks) == 0 {
		t.Fatal(
			"expected the default fixture to seed drivers, assets, addresses, routes and webhooks",
		)
	}

	for _, record := range drivers {
		mustNumericID(t, "driver", recordID(record))
	}
	for _, record := range addresses {
		mustNumericID(t, "address", recordID(record))
	}
	for _, record := range routes {
		mustNumericID(t, "route", recordID(record))
	}
	for _, record := range webhooks {
		mustNumericID(t, "webhook", recordID(record))
	}
	for _, record := range assets {
		if value := mustNumericID(t, "asset", recordID(record)); value < 1<<48 {
			t.Fatalf("expected asset id %d in the Samsara device range", value)
		}
	}

	driverIDs := recordIDSet(drivers)
	assetIDs := recordIDSet(assets)
	for _, route := range routes {
		if _, ok := driverIDs[nestedString(route, "driver", "id")]; !ok {
			t.Fatalf(
				"route %s references unknown driver %q",
				recordID(route),
				nestedString(route, "driver", "id"),
			)
		}
		if _, ok := assetIDs[nestedString(route, "vehicle", "id")]; !ok {
			t.Fatalf(
				"route %s references unknown vehicle %q",
				recordID(route),
				nestedString(route, "vehicle", "id"),
			)
		}
	}
	for _, record := range list(ResourceVehicleStats) {
		if _, ok := assetIDs[recordID(record)]; !ok {
			t.Fatalf("vehicle stats reference unknown vehicle %q", recordID(record))
		}
	}
	for _, record := range list(ResourceAssetLocation) {
		if _, ok := assetIDs[nestedString(record, "asset", "id")]; !ok {
			t.Fatalf(
				"asset location references unknown asset %q",
				nestedString(record, "asset", "id"),
			)
		}
	}
	for _, record := range list(ResourceHOSClocks) {
		if _, ok := driverIDs[nestedString(record, "driver", "id")]; !ok {
			t.Fatalf("hos clock references unknown driver %q", nestedString(record, "driver", "id"))
		}
		if vehicleID := nestedString(record, "currentVehicle", "id"); vehicleID != "" {
			if _, ok := assetIDs[vehicleID]; !ok {
				t.Fatalf("hos clock references unknown vehicle %q", vehicleID)
			}
		}
	}

	for _, record := range list(ResourceFormTemplates) {
		if !uuidPattern.MatchString(recordID(record)) {
			t.Fatalf("expected form template uuid, got %q", recordID(record))
		}
		mustNumericID(t, "form template author", nestedString(record, "createdBy", "id"))
	}
	for _, record := range list(ResourceFormSubmissions) {
		if !uuidPattern.MatchString(recordID(record)) {
			t.Fatalf("expected form submission uuid, got %q", recordID(record))
		}
	}
	for _, record := range list(ResourceLiveShares) {
		if !liveShareIDPattern.MatchString(recordID(record)) {
			t.Fatalf(
				"expected live share id of 19 lowercase alphanumerics, got %q",
				recordID(record),
			)
		}
	}
	for _, resource := range []Resource{ResourceDriverTachograph, ResourceVehicleTachograph} {
		for _, record := range list(resource) {
			files, ok := record["files"].([]any)
			if !ok || len(files) == 0 {
				t.Fatalf("expected tachograph files on %s record", resource)
			}
			for _, raw := range files {
				file, okFile := anyAsMap(raw)
				if !okFile || !uuidPattern.MatchString(stringValue(Record(file), "id")) {
					t.Fatalf("expected tachograph file uuid, got %v", raw)
				}
			}
		}
	}

	messages := list(ResourceMessages)
	if len(messages) == 0 {
		t.Fatal("expected seeded driver messages")
	}
	for _, message := range messages {
		driverID, ok := message["driverId"].(float64)
		if !ok || driverID != float64(int64(driverID)) {
			t.Fatalf("expected integer message driverId, got %v", message["driverId"])
		}
		if _, known := driverIDs[strconv.FormatInt(int64(driverID), 10)]; !known {
			t.Fatalf("message driverId %d does not reference a fixture driver", int64(driverID))
		}
	}
}

func TestStoreGeneratesNumericIDsAboveExistingIDs(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{
		Addresses: []Record{{"id": "41226316"}, {"id": "41226405"}},
		Drivers:   []Record{{"id": testDriverID}},
		Routes:    []Record{{"id": "4129806626"}},
		Webhooks:  []Record{{"id": "523918"}},
		Assets:    []Record{{"id": fixtureVehicleID, "type": "vehicle"}},
		VehicleStats: []Record{
			{"id": "281474977099990"},
		},
	})

	cases := []struct {
		resource Resource
		floor    int64
	}{
		{resource: ResourceAddresses, floor: 41226405},
		{resource: ResourceDrivers, floor: 1654973},
		{resource: ResourceRoutes, floor: 4129806626},
		{resource: ResourceWebhooks, floor: 523918},
		{resource: ResourceAssets, floor: 281474977099990},
	}
	for _, tc := range cases {
		first, err := store.Create(tc.resource, Record{"name": "first"})
		if err != nil {
			t.Fatalf("create %s: %v", tc.resource, err)
		}
		second, err := store.Create(tc.resource, Record{"name": "second"})
		if err != nil {
			t.Fatalf("create %s: %v", tc.resource, err)
		}
		firstID := mustNumericID(t, string(tc.resource), recordID(first))
		secondID := mustNumericID(t, string(tc.resource), recordID(second))
		if firstID <= tc.floor {
			t.Fatalf("expected %s id above %d, got %d", tc.resource, tc.floor, firstID)
		}
		if secondID <= firstID {
			t.Fatalf("expected increasing %s ids, got %d then %d", tc.resource, firstID, secondID)
		}
	}

	explicit, err := store.Create(ResourceAssets, Record{"id": "281474980000000"})
	if err != nil {
		t.Fatalf("create asset with explicit id: %v", err)
	}
	next, err := store.Create(ResourceAssets, Record{"name": "after explicit"})
	if err != nil {
		t.Fatalf("create asset after explicit id: %v", err)
	}
	if mustNumericID(t, "asset", recordID(next)) <= mustNumericID(t, "asset", recordID(explicit)) {
		t.Fatalf("expected generated asset id above %s, got %s", recordID(explicit), recordID(next))
	}
}

func TestStoreGeneratedIDsAreDeterministicAcrossReset(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{Drivers: []Record{{"id": testDriverID}}})
	resources := []Resource{
		ResourceDrivers,
		ResourceAddresses,
		ResourceFormSubmissions,
		ResourceLiveShares,
	}
	before := make([]string, 0, len(resources))
	for _, resource := range resources {
		created, err := store.Create(resource, Record{"name": "first"})
		if err != nil {
			t.Fatalf("create %s: %v", resource, err)
		}
		before = append(before, recordID(created))
	}

	store.Reset()
	for idx, resource := range resources {
		created, err := store.Create(resource, Record{"name": "first"})
		if err != nil {
			t.Fatalf("create %s after reset: %v", resource, err)
		}
		if recordID(created) != before[idx] {
			t.Fatalf(
				"expected %s id %q after reset, got %q",
				resource,
				before[idx],
				recordID(created),
			)
		}
	}
}

func TestStoreGeneratesUUIDAndLiveShareIDs(t *testing.T) {
	t.Parallel()

	existingSubmission := generatedRecordID(idSpaceFormSubmissions, idKindUUID, 1)
	existingShare := generatedRecordID(idSpaceLiveShares, idKindToken, 1)
	store := NewStore(&Fixture{
		FormSubmissions: []Record{{"id": existingSubmission}},
		LiveShares:      []Record{{"id": existingShare}},
	})

	seenSubmissions := map[string]struct{}{existingSubmission: {}}
	seenShares := map[string]struct{}{existingShare: {}}
	for idx := 0; idx < 5; idx++ {
		submission, err := store.Create(ResourceFormSubmissions, Record{"title": "generated"})
		if err != nil {
			t.Fatalf("create form submission: %v", err)
		}
		submissionID := recordID(submission)
		if !uuidPattern.MatchString(submissionID) {
			t.Fatalf("expected generated form submission uuid, got %q", submissionID)
		}
		if _, dup := seenSubmissions[submissionID]; dup {
			t.Fatalf("expected unique form submission id, got duplicate %q", submissionID)
		}
		seenSubmissions[submissionID] = struct{}{}

		share, err := store.Create(ResourceLiveShares, Record{"name": "generated"})
		if err != nil {
			t.Fatalf("create live share: %v", err)
		}
		shareID := recordID(share)
		if !liveShareIDPattern.MatchString(shareID) {
			t.Fatalf("expected generated live share id, got %q", shareID)
		}
		if _, dup := seenShares[shareID]; dup {
			t.Fatalf("expected unique live share id, got duplicate %q", shareID)
		}
		seenShares[shareID] = struct{}{}
	}
}

func TestStoreDoesNotAssignIDsToKeyedResources(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{})
	created, err := store.Create(ResourceAssetLocation, Record{
		"asset": map[string]any{"id": fixtureVehicleID},
	})
	if err != nil {
		t.Fatalf("create asset location: %v", err)
	}
	if _, hasID := created["id"]; hasID {
		t.Fatalf("expected asset location record without id, got %v", created["id"])
	}
}

func TestDerivedResourceIDsAreNumericAndStable(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	previousStop := int64(0)
	for sequence := 1; sequence <= 6; sequence++ {
		stopID := routeStopID(fixtureRouteID, sequence)
		stopValue := mustNumericID(t, "route stop", stopID)
		if previousStop != 0 && stopValue != previousStop+1 {
			t.Fatalf(
				"expected consecutive stop ids on a route, got %d after %d",
				stopValue,
				previousStop,
			)
		}
		previousStop = stopValue
		if stopID != routeStopID(fixtureRouteID, sequence) {
			t.Fatalf("expected stable route stop id for sequence %d", sequence)
		}
		if previous, dup := seen[stopID]; dup {
			t.Fatalf("route stop %d and %d share id %q", previous, sequence, stopID)
		}
		seen[stopID] = sequence
	}
	if routeStopID(fixtureRouteID, 1) == routeStopID("4129806458", 1) {
		t.Fatal("expected route stop ids to differ between routes")
	}
	mustNumericID(t, "address route stop", routeStopIDForAddress(fixtureRouteID, "41226316"))

	day := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC).Format("20060102")
	pre := dvirRecordID(day, testDriverID, "pre")
	post := dvirRecordID(day, testDriverID, "post")
	mustNumericID(t, "dvir", pre)
	mustNumericID(t, "dvir", post)
	if pre == post {
		t.Fatal("expected pre-trip and post-trip DVIRs to have distinct ids")
	}
	if dvirDefectID(pre, 1) == dvirDefectID(pre, 2) {
		t.Fatal("expected distinct DVIR defect ids")
	}
	mustNumericID(t, "dvir defect", dvirDefectID(pre, 1))

	submissionID := formSubmissionID(
		time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		testDriverID,
		"bol",
	)
	if !uuidPattern.MatchString(submissionID) {
		t.Fatalf("expected generated form submission uuid, got %q", submissionID)
	}
}

func TestLegacyMessagesUseIntegerDriverIDs(t *testing.T) {
	t.Parallel()

	srv := newDefaultFixtureFormServer(t)
	listed := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/v1/fleet/messages?endMs=1772460000000&durationMs=172800000",
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("expected 200 listing messages, got %d", listed.Code)
	}
	if !bytes.Contains(listed.Body.Bytes(), []byte(`"driverId":1654973`)) {
		t.Fatalf(
			"expected seeded message for driver 1654973 as an integer, got %s",
			listed.Body.String(),
		)
	}

	created := performAuthorizedRequestWithBody(
		srv,
		http.MethodPost,
		"/v1/fleet/messages",
		map[string]any{
			"driverIds": []int64{1654988},
			"text":      "Confirm trailer seal before departure.",
		},
	)
	if created.Code != http.StatusOK {
		t.Fatalf("expected 200 creating message, got %d: %s", created.Code, created.Body.String())
	}
	if !bytes.Contains(created.Body.Bytes(), []byte(`"driverId":1654988`)) {
		t.Fatalf("expected created message to echo integer driverId, got %s", created.Body.String())
	}

	relisted := performAuthorizedRequest(srv, http.MethodGet, "/v1/fleet/messages")
	if !bytes.Contains(relisted.Body.Bytes(), []byte(`"driverId":1654988`)) {
		t.Fatalf("expected stored message for driver 1654988, got %s", relisted.Body.String())
	}
}
