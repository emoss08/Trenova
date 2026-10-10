package sim

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

func TestAssetListFiltersAndIncludes(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	all := requireStatus(t, srv, http.MethodGet, "/assets", nil, http.StatusOK).list(t)
	if len(all) != 31 {
		t.Fatalf("expected 12 tractors, 14 trailers and 5 equipment units, got %d", len(all))
	}
	for _, asset := range all {
		for _, hidden := range []string{fieldExternalIDs, "tags", fieldAttributesKey} {
			if _, has := asset[hidden]; has {
				t.Fatalf("expected %s only on request, got it on %s", hidden, recordID(asset))
			}
		}
		for key := range asset {
			if len(key) > 3 && key[:3] == "sim" {
				t.Fatalf("expected internal fields never to leak, got %s", key)
			}
		}
	}
	included := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/assets?includeExternalIds=true&includeTags=true&includeAttributes=true&type=vehicle",
		nil,
		http.StatusOK,
	).list(t)
	if len(included) != 12 || nestedString(included[0], fieldExternalIDs, "tmsVehicleId") == "" ||
		len(tagIDsOf(t, included[0])) == 0 || len(included[0][fieldAttributesKey].([]any)) == 0 {
		t.Fatalf("expected includes to add externalIds, tags and attributes, got %v", included[0])
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/assets?includeTags=yes",
		nil,
	).expectError(t, http.StatusBadRequest, "includeTags")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/assets?type=truck",
		nil,
	).expectError(t, http.StatusBadRequest, "type")

	byIDs := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/assets?ids="+fixtureTruck1001+",tmsTrailerId:trailer-2042,samsara.vin:4V4NC9EH6EN171002",
		nil,
		http.StatusOK,
	).list(t)
	got := listIDs(byIDs)
	slices.Sort(got)
	want := []string{fixtureTruck1001, fixtureTruck1002, fixtureTrailer2042}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("expected ids to accept IDs and external IDs, got %v", got)
	}
	byExternal := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/assets?externalIds=tmsTrailerId:trailer-2041,tmsEquipmentId:equip-yard-spotter-01",
		nil,
		http.StatusOK,
	).list(t)
	if len(byExternal) != 2 {
		t.Fatalf("expected two assets by external ID, got %v", listIDs(byExternal))
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/assets?externalIds=plain",
		nil,
	).expectError(t, http.StatusBadRequest, "key:value")

	long := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/assets",
			map[string]string{"attributes": "Trailer Length (ft):range(50,)"},
		),
		nil,
		http.StatusOK,
	).list(t)
	if len(long) != 14 {
		t.Fatalf("expected the 53ft trailers, got %d", len(long))
	}
	callAPI(
		t,
		srv,
		http.MethodGet,
		query("/assets", map[string]string{"attributes": "Trailer Type:Reefer"}),
		nil,
	).
		expectError(t, http.StatusBadRequest, "range")
	callAPI(
		t,
		srv,
		http.MethodGet,
		query("/assets", map[string]string{"attributes": "Length:range(,)"}),
		nil,
	).
		expectError(t, http.StatusBadRequest, "bound")
	updated := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/assets?updatedAfterTime=2026-02-20T00:00:00Z",
		nil,
		http.StatusOK,
	).list(t)
	for _, asset := range updated {
		if stringValue(asset, "updatedAtTime") < "2026-02-20T00:00:00Z" {
			t.Fatalf("expected only recently updated assets, got %v", asset)
		}
	}
	if len(updated) == 0 || len(updated) == len(all) {
		t.Fatalf("expected updatedAfterTime to narrow the list, got %d", len(updated))
	}
}

func TestAssetWritesAndVehicleWebhooks(t *testing.T) {
	t.Parallel()

	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true, WebhookURL: sink.url})
	for _, testCase := range []struct {
		body     map[string]any
		fragment string
	}{
		{body: map[string]any{"type": "truck"}, fragment: "type"},
		{body: map[string]any{"regulationMode": "maybe"}, fragment: "regulationMode"},
		{body: map[string]any{"year": float64(1800)}, fragment: "year"},
		{body: map[string]any{"year": float64(2020.5)}, fragment: "integer"},
		{body: map[string]any{"vin": "1FUJGLDR5CLBP1001"}, fragment: "already assigned"},
		{body: map[string]any{"externalIds": map[string]any{"tmsVehicleId": "unit-1001"}}, fragment: "already assigned"},
		{body: map[string]any{"tagIds": []any{"1"}}, fragment: "tag"},
		{body: map[string]any{"attributes": []any{map[string]any{"name": "Length"}}}, fragment: "values"},
	} {
		callAPI(
			t,
			srv,
			http.MethodPost,
			"/assets",
			testCase.body,
		).expectError(t, http.StatusBadRequest, testCase.fragment)
	}

	trailer := requireStatus(t, srv, http.MethodPost, "/assets", map[string]any{
		"name": "Container 7", "type": "unpowered", "tagIds": []any{tagAustinTerminal},
	}, http.StatusOK).data(t)
	uncategorized := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/assets",
		map[string]any{"name": "Pallet Jack"},
		http.StatusOK,
	).data(t)
	if uncategorized["type"] != assetTypeUncategorized {
		t.Fatalf("expected the default type, got %v", uncategorized["type"])
	}
	vehicle := requireStatus(t, srv, http.MethodPost, "/assets", map[string]any{
		"name": "Truck 2001", "type": "vehicle", "vin": "1FUJGLDR5CLBP2001", "year": float64(2024),
		"tagIds": []any{tagTractors}, "externalIds": map[string]any{"tmsVehicleId": "unit-2001"},
		"attributes": []any{map[string]any{"name": "Fuel Type", "stringValues": []any{"Diesel"}}},
	}, http.StatusOK).data(t)
	vehicleID := recordID(vehicle)
	if !slices.Equal(tagIDsOf(t, vehicle), []string{tagTractors}) ||
		vehicle["year"] != float64(2024) {
		t.Fatalf("unexpected created vehicle %v", vehicle)
	}
	tractors := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/"+tagTractors,
		nil,
		http.StatusOK,
	).data(t)
	if !slices.Contains(listIDs(recordsFromAny(tractors["vehicles"])), vehicleID) {
		t.Fatal("expected the new vehicle under the tag's vehicles")
	}
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/assets?id="+recordID(trailer),
		map[string]any{"notes": "dock 4"},
		http.StatusOK,
	)
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/assets?id=tmsVehicleId:unit-2001",
		map[string]any{"notes": "spare"},
		http.StatusOK,
	)

	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/assets?id="+fixtureTruck1001,
		map[string]any{"make": "Mack"},
	).
		expectError(t, http.StatusBadRequest, "gateway")
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/assets?id="+fixtureTruck1001,
		map[string]any{"make": "Freightliner"},
		http.StatusOK,
	)
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/assets",
		map[string]any{"name": "x"},
	).expectError(t, http.StatusBadRequest, "id")
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/assets?id=999",
		map[string]any{"name": "x"},
	).expectError(t, http.StatusNotFound, "asset")

	moved := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/assets?id="+vehicleID,
		map[string]any{"type": "equipment"},
		http.StatusOK,
	).data(t)
	if moved["type"] != assetTypeEquipment ||
		!slices.Equal(tagIDsOf(t, moved), []string{tagTractors}) {
		t.Fatalf("expected the tag membership to follow the type change, got %v", moved)
	}
	tractors = requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/"+tagTractors,
		nil,
		http.StatusOK,
	).data(t)
	if !slices.Contains(listIDs(recordsFromAny(tractors["assets"])), vehicleID) ||
		slices.Contains(listIDs(recordsFromAny(tractors["vehicles"])), vehicleID) {
		t.Fatal("expected the retyped asset to move from vehicles to assets on the tag")
	}

	created := waitForWebhookEvents(t, sink, eventVehicleCreated, 1)
	updated := waitForWebhookEvents(t, sink, eventVehicleUpdated, 2)
	time.Sleep(100 * time.Millisecond)
	if len(waitForWebhookEvents(t, sink, eventVehicleCreated, 1)) != 1 {
		t.Fatal("expected VehicleCreated only for the vehicle asset")
	}
	if len(waitForWebhookEvents(t, sink, eventVehicleUpdated, 2)) != 2 {
		t.Fatal("expected VehicleUpdated only for vehicle-typed assets after the change")
	}
	for _, event := range append(created, updated...) {
		data := webhookData(t, event)
		if _, ok := anyAsMap(data["vehicle"]); !ok {
			t.Fatalf("expected a {vehicle} payload, got %v", data)
		}
	}

	callAPI(
		t,
		srv,
		http.MethodDelete,
		"/assets?id=tmsVehicleId:unit-1002",
		nil,
	).expectError(t, http.StatusNotFound, "asset")
	before := requireStatus(t, srv, http.MethodGet, "/fleet/trailers/stats?types=gps&trailerIds="+fixtureTrailer2042, nil, http.StatusOK).list(t)[0]
	requireStatus(
		t,
		srv,
		http.MethodDelete,
		"/assets?id="+fixtureTruck1002,
		nil,
		http.StatusNoContent,
	)
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/"+fixtureTruck1002,
		nil,
	).expectError(t, http.StatusNotFound, "vehicle")
	srv.clock.Step(time.Hour)
	after := requireStatus(t, srv, http.MethodGet, "/fleet/trailers/stats?types=gps&trailerIds="+fixtureTrailer2042, nil, http.StatusOK).list(t)[0]
	if floatFromAny(
		nestedAny(after, "gps", "latitude"),
	) != floatFromAny(
		nestedAny(before, "gps", "latitude"),
	) ||
		floatFromAny(nestedAny(after, "gps", "speedMilesPerHour")) != 0 {
		t.Fatalf(
			"expected the uncoupled trailer to stay where the tractor left it, got %v vs %v",
			before["gps"],
			after["gps"],
		)
	}
	time.Sleep(100 * time.Millisecond)
	for _, event := range sink.snapshot() {
		if event.EventType == "VehicleDeleted" {
			t.Fatal("expected no delete webhook: the spec has no VehicleDeleted event type")
		}
	}
}
