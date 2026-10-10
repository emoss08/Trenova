package sim

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func austinCircleAddress(name string) map[string]any {
	return map[string]any{
		"name":             name,
		"formattedAddress": "7000 Burleson Rd, Austin, TX 78744",
		"geofence": map[string]any{
			"circle": map[string]any{"radiusMeters": float64(150)},
		},
	}
}

func squarePolygon(latitude, longitude, half float64) map[string]any {
	return map[string]any{
		"vertices": []any{
			map[string]any{"latitude": latitude - half, "longitude": longitude - half},
			map[string]any{"latitude": latitude - half, "longitude": longitude + half},
			map[string]any{"latitude": latitude + half, "longitude": longitude + half},
			map[string]any{"latitude": latitude + half, "longitude": longitude - half},
		},
	}
}

func TestAddressListRendersTagsContactsAndFilters(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	all := requireStatus(t, srv, http.MethodGet, "/addresses", nil, http.StatusOK).list(t)
	if len(all) != 7 {
		t.Fatalf("expected 7 fixture addresses, got %d", len(all))
	}
	yard := requireRecord(t, all, fixtureAustinYard)
	tags := listOf(yard["tags"])
	if len(tags) != 1 || stringValue(Record(mapOf(tags[0])), "id") != tagAustinTerminal ||
		stringValue(Record(mapOf(tags[0])), "parentTagId") != tagCentralTexas {
		t.Fatalf("expected Austin Terminal tag with parent, got %v", yard["tags"])
	}
	contacts := listOf(yard["contacts"])
	if len(contacts) != 1 || stringValue(Record(mapOf(contacts[0])), "firstName") != "Marisol" {
		t.Fatalf("expected the yard contact mini-object, got %v", yard["contacts"])
	}
	if types := stringListValues(yard["addressTypes"]); len(types) != 1 || types[0] != "yard" {
		t.Fatalf("expected addressTypes [yard], got %v", yard["addressTypes"])
	}
	if stringValue(yard, "createdAtTime") == "" {
		t.Fatal("expected createdAtTime on the fixture address")
	}

	byTag := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/addresses",
			map[string]string{"tagIds": tagAustinTerminal},
		),
		nil,
		http.StatusOK,
	).list(t)
	if len(byTag) != 2 {
		t.Fatalf("expected 2 Austin Terminal addresses, got %v", listIDs(byTag))
	}
	byParent := requireStatus(
		t,
		srv,
		http.MethodGet,
		query(
			"/addresses",
			map[string]string{"parentTagIds": tagCentralTexas},
		),
		nil,
		http.StatusOK,
	).list(t)
	if len(byParent) != 3 {
		t.Fatalf(
			"expected Central Texas descendants to hold 3 addresses, got %v",
			listIDs(byParent),
		)
	}
	recent := requireStatus(t, srv, http.MethodGet,
		query("/addresses", map[string]string{"createdAfterTime": "2025-12-01T19:30:00.000-00:00"}),
		nil, http.StatusOK).list(t)
	if len(recent) != 3 {
		t.Fatalf("expected 3 addresses created at or after Dec 1, got %v", listIDs(recent))
	}
	callAPI(t, srv, http.MethodGet, "/addresses?createdAfterTime=yesterday", nil).
		expectError(t, http.StatusBadRequest, "createdAfterTime")

	first := requireStatus(t, srv, http.MethodGet, "/addresses?limit=4", nil, http.StatusOK)
	if len(first.list(t)) != 4 || first.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a 4-address first page, got %s", first.Body)
	}
	cursor := stringValue(first.pagination(t), "endCursor")
	second := requireStatus(t, srv, http.MethodGet,
		query("/addresses", map[string]string{"limit": "4", "after": cursor}), nil, http.StatusOK)
	if len(second.list(t)) != 3 || second.pagination(t)["hasNextPage"] != false {
		t.Fatalf("expected the remaining 3 addresses, got %s", second.Body)
	}
}

func TestAddressCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"externalIds": map[string]any{"siteId": "AUS77"},
	}).expect(t, http.StatusOK)

	with := func(mutate func(body map[string]any)) map[string]any {
		body := austinCircleAddress("Validation Yard")
		mutate(body)
		return body
	}
	geofence := func(value any) map[string]any {
		return with(func(body map[string]any) { body["geofence"] = value })
	}
	vertices := func(points ...[2]float64) map[string]any {
		items := make([]any, 0, len(points))
		for _, point := range points {
			items = append(items, map[string]any{"latitude": point[0], "longitude": point[1]})
		}
		return geofence(map[string]any{"polygon": map[string]any{"vertices": items}})
	}
	manyVertices := make([][2]float64, 0, 41)
	for idx := 0; idx < 41; idx++ {
		manyVertices = append(
			manyVertices,
			[2]float64{30 + float64(idx)*0.0001, -97 + float64(idx%2)*0.001},
		)
	}

	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{"missing name", with(func(b map[string]any) { delete(b, "name") }), "name is required"},
		{"blank name", with(func(b map[string]any) { b["name"] = "  " }), "name cannot be blank"},
		{
			"long name",
			with(func(b map[string]any) { b["name"] = strings.Repeat("n", 256) }),
			"at most 255",
		},
		{
			"missing formatted",
			with(func(b map[string]any) { delete(b, "formattedAddress") }),
			"formattedAddress is required",
		},
		{
			"long formatted",
			with(func(b map[string]any) { b["formattedAddress"] = strings.Repeat("f", 1025) }),
			"at most 1024",
		},
		{
			"long notes",
			with(func(b map[string]any) { b["notes"] = strings.Repeat("x", 281) }),
			"at most 280",
		},
		{
			"missing geofence",
			with(func(b map[string]any) { delete(b, "geofence") }),
			"geofence is required",
		},
		{"empty geofence", geofence(map[string]any{}), "exactly one of circle or polygon"},
		{
			"both shapes",
			geofence(map[string]any{
				"circle":  map[string]any{"radiusMeters": float64(50)},
				"polygon": squarePolygon(30.2, -97.7, 0.01),
			}),
			"exactly one of circle or polygon",
		},
		{
			"missing radius",
			geofence(map[string]any{"circle": map[string]any{}}),
			"radiusMeters is required",
		},
		{
			"zero radius",
			geofence(map[string]any{"circle": map[string]any{"radiusMeters": float64(0)}}),
			"between 1",
		},
		{
			"fractional radius",
			geofence(map[string]any{"circle": map[string]any{"radiusMeters": 12.5}}),
			"integer",
		},
		{
			"latitude without longitude",
			geofence(
				map[string]any{
					"circle": map[string]any{"radiusMeters": float64(50), "latitude": 30.1},
				},
			),
			"provided together",
		},
		{
			"circle latitude range",
			geofence(map[string]any{"circle": map[string]any{
				"radiusMeters": float64(50), "latitude": 91.0, "longitude": -97.0,
			}}),
			"geofence.circle.latitude",
		},
		{"two vertices", vertices([2]float64{30, -97}, [2]float64{30.1, -97}), "3 to 40"},
		{"forty-one vertices", vertices(manyVertices...), "3 to 40"},
		{
			"vertex longitude missing",
			geofence(map[string]any{"polygon": map[string]any{"vertices": []any{
				map[string]any{"latitude": 30.0, "longitude": -97.0},
				map[string]any{"latitude": 30.1, "longitude": -97.0},
				map[string]any{"latitude": 30.1},
			}}}),
			"vertices[2].longitude is required",
		},
		{
			"vertex range",
			vertices([2]float64{30, -97}, [2]float64{30.1, -197}, [2]float64{30.2, -97.1}),
			"longitude",
		},
		{
			"collinear",
			vertices([2]float64{30, -97}, [2]float64{30.1, -97}, [2]float64{30.2, -97}),
			"enclose an area",
		},
		{
			"repeated vertex",
			vertices(
				[2]float64{30, -97},
				[2]float64{30.1, -97},
				[2]float64{30, -97},
				[2]float64{30.1, -97.1},
			),
			"repeat a vertex",
		},
		{
			"bow tie",
			vertices(
				[2]float64{30, -97},
				[2]float64{30.1, -96.9},
				[2]float64{30.1, -97},
				[2]float64{30, -96.9},
			),
			"must not cross",
		},
		{"bad settings", geofence(map[string]any{
			"circle":   map[string]any{"radiusMeters": float64(50)},
			"settings": map[string]any{"showAddresses": "yes"},
		}), "showAddresses"},
		{
			"address type",
			with(func(b map[string]any) { b["addressTypes"] = []any{"warehouse"} }),
			"addressTypes",
		},
		{"unknown tag", with(func(b map[string]any) { b["tagIds"] = []any{"999999"} }), "tag"},
		{
			"unknown contact",
			with(func(b map[string]any) { b["contactIds"] = []any{"1"} }),
			"contact",
		},
		{
			"external id taken by a driver",
			with(func(b map[string]any) { b["externalIds"] = map[string]any{"siteId": "AUS77"} }),
			"unique across all objects",
		},
		{
			"ungeocodable",
			with(func(b map[string]any) { b["formattedAddress"] = "1 Unknown Way, Nowhere, ZZ" }),
			"could not be geocoded",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/addresses", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}

	listed := requireStatus(t, srv, http.MethodGet, "/addresses", nil, http.StatusOK).list(t)
	if len(listed) != 7 {
		t.Fatalf("expected failed creates to leave 7 addresses, got %d", len(listed))
	}
}

func TestAddressLifecycleWithWebhooks(t *testing.T) {
	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{WebhookURL: sink.url})

	body := austinCircleAddress("Southeast Austin Cross-Dock")
	body["notes"] = "Dock doors 4-12; check in at guard shack"
	body["addressTypes"] = []any{"customerSite", "yard", "customerSite"}
	body["contactIds"] = []any{"22414"}
	body["tagIds"] = []any{tagAustinTerminal}
	body["externalIds"] = map[string]any{"siteId": "SEAUS1", "erpCode": float64(4412)}
	body["geofence"].(map[string]any)["settings"] = map[string]any{"showAddresses": true}
	created := requireStatus(t, srv, http.MethodPost, "/addresses", body, http.StatusOK).data(t)
	addressID := recordID(created)
	if mustNumericID(t, "address", addressID) <= 41226405 {
		t.Fatalf("expected a new address ID above the fixture range, got %s", addressID)
	}
	if created["createdAtTime"] != fleetTestTime.Format("2006-01-02T15:04:05Z07:00") {
		t.Fatalf("expected createdAtTime from the sim clock, got %v", created["createdAtTime"])
	}
	circle := mapOf(mapOf(created["geofence"])["circle"])
	latitude, _ := circle["latitude"].(float64)
	longitude, _ := circle["longitude"].(float64)
	if haversineMeters(latitude, longitude, 30.2672, -97.7431) > 6000 {
		t.Fatalf("expected the circle geocoded near Austin, got %v,%v", latitude, longitude)
	}
	if created["latitude"] != latitude || created["longitude"] != longitude {
		t.Fatalf("expected address coordinates to match the geocoded circle, got %v", created)
	}
	if mapOf(mapOf(created["geofence"])["settings"])["showAddresses"] != true {
		t.Fatalf("expected geofence settings to round-trip, got %v", created["geofence"])
	}
	if types := stringListValues(
		created["addressTypes"],
	); strings.Join(
		types,
		",",
	) != "customerSite,yard" {
		t.Fatalf("expected de-duplicated addressTypes, got %v", types)
	}
	if ids := mapOf(created["externalIds"]); ids["siteId"] != "SEAUS1" || ids["erpCode"] != "4412" {
		t.Fatalf("expected stringified external IDs, got %v", ids)
	}
	if contacts := listOf(created["contacts"]); len(contacts) != 1 ||
		mapOf(contacts[0])["lastName"] != "Holloway" {
		t.Fatalf("expected the contact mini-object, got %v", created["contacts"])
	}

	tag := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/"+tagAustinTerminal,
		nil,
		http.StatusOK,
	).data(t)
	if _, ok := idSet(recordsOf(listOf(tag["addresses"])))[addressID]; !ok {
		t.Fatalf("expected the tag to list the new address, got %v", tag["addresses"])
	}

	events := waitForWebhookEvents(t, sink, "AddressCreated", 1)
	address := mapOf(webhookData(t, events[0])["address"])
	if address == nil || stringValue(Record(address), "id") != addressID ||
		stringValue(Record(address), "formattedAddress") == "" || address["geofence"] == nil {
		t.Fatalf("expected AddressCreated to carry {address: <Address>}, got %v", events[0].Data)
	}

	externalRef := "/addresses/" + url.PathEscape("siteId:SEAUS1")
	fetched := requireStatus(t, srv, http.MethodGet, externalRef, nil, http.StatusOK).data(t)
	if recordID(fetched) != addressID {
		t.Fatalf("expected the external ID lookup to resolve, got %v", fetched)
	}

	patched := requireStatus(t, srv, http.MethodPatch, externalRef, map[string]any{
		"notes":            nil,
		"formattedAddress": "1200 Commerce St, Dallas, TX 75202",
		"geofence":         map[string]any{"polygon": squarePolygon(32.78, -96.80, 0.002)},
		"tagIds":           []any{tagDFWTerminal},
		"contactIds":       []any{},
	}, http.StatusOK).data(t)
	if _, has := patched["notes"]; has {
		t.Fatalf("expected notes cleared, got %v", patched["notes"])
	}
	if _, has := mapOf(patched["geofence"])["circle"]; has {
		t.Fatalf("expected the polygon to replace the circle, got %v", patched["geofence"])
	}
	if vertices := listOf(
		mapOf(mapOf(patched["geofence"])["polygon"])["vertices"],
	); len(
		vertices,
	) != 4 {
		t.Fatalf("expected 4 polygon vertices, got %v", patched["geofence"])
	}
	patchedLatitude, _ := patched["latitude"].(float64)
	patchedLongitude, _ := patched["longitude"].(float64)
	if haversineMeters(patchedLatitude, patchedLongitude, 32.7767, -96.7970) > 6000 {
		t.Fatalf("expected the new formattedAddress to be geocoded near Dallas, got %v", patched)
	}
	if tags := listOf(patched["tags"]); len(tags) != 1 || mapOf(tags[0])["id"] != tagDFWTerminal {
		t.Fatalf("expected tags replaced by DFW Terminal, got %v", patched["tags"])
	}
	if len(listOf(patched["contacts"])) != 0 ||
		patched["createdAtTime"] != created["createdAtTime"] {
		t.Fatalf("expected contacts cleared and createdAtTime kept, got %v", patched)
	}
	if _, has := patched["updatedAtTime"]; has {
		t.Fatal("expected no updatedAtTime on addresses")
	}
	updates := waitForWebhookEvents(t, sink, "AddressUpdated", 1)
	if polygon := mapOf(
		mapOf(mapOf(webhookData(t, updates[0])["address"])["geofence"])["polygon"],
	); polygon == nil {
		t.Fatalf("expected AddressUpdated to carry the new polygon, got %v", updates[0].Data)
	}

	requireStatus(t, srv, http.MethodPatch, "/addresses/"+addressID, map[string]any{
		"geofence": nil,
	}, http.StatusBadRequest)
	requireStatus(t, srv, http.MethodPatch, "/addresses/"+addressID, map[string]any{
		"latitude": 32.7,
	}, http.StatusBadRequest)
	requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/addresses/999",
		map[string]any{"name": "x"},
		http.StatusNotFound,
	)

	request := callAPI(t, srv, http.MethodDelete, externalRef, nil)
	request.expect(t, http.StatusNoContent)
	deletions := waitForWebhookEvents(t, sink, "AddressDeleted", 1)
	minified := mapOf(webhookData(t, deletions[0])["address"])
	if stringValue(Record(minified), "id") != addressID ||
		stringValue(Record(minified), "name") != "Southeast Austin Cross-Dock" ||
		mapOf(minified["externalIds"])["siteId"] != "SEAUS1" || len(minified) != 3 {
		t.Fatalf("expected the minified AddressDeleted payload, got %v", minified)
	}
	requireStatus(t, srv, http.MethodGet, "/addresses/"+addressID, nil, http.StatusNotFound)
	dfw := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/"+tagDFWTerminal,
		nil,
		http.StatusOK,
	).data(t)
	if _, still := idSet(recordsOf(listOf(dfw["addresses"])))[addressID]; still {
		t.Fatalf("expected the deleted address to leave its tags, got %v", dfw["addresses"])
	}
	requireStatus(t, srv, http.MethodDelete, "/addresses/"+addressID, nil, http.StatusNotFound)
}

func TestAddressPolygonDefaultsToCentroidAndDropsClosingVertex(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	polygon := squarePolygon(31.5493, -97.1467, 0.004)
	vertices := polygon["vertices"].([]any)
	polygon["vertices"] = append(vertices, vertices[0])
	created := requireStatus(t, srv, http.MethodPost, "/addresses", map[string]any{
		"name":             "Waco Relay Lot",
		"formattedAddress": "Relay Lot, Unlisted Township",
		"geofence":         map[string]any{"polygon": polygon},
	}, http.StatusOK).data(t)
	if got := len(listOf(mapOf(mapOf(created["geofence"])["polygon"])["vertices"])); got != 4 {
		t.Fatalf("expected the closing vertex to be dropped, got %d vertices", got)
	}
	latitude, _ := created["latitude"].(float64)
	longitude, _ := created["longitude"].(float64)
	if haversineMeters(latitude, longitude, 31.5493, -97.1467) > 5 {
		t.Fatalf("expected the centroid as the address location, got %v,%v", latitude, longitude)
	}
}

func TestContactsCRUDAndAddressDetach(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	listed := requireStatus(t, srv, http.MethodGet, "/contacts", nil, http.StatusOK).list(t)
	if len(listed) != 4 {
		t.Fatalf("expected 4 fixture contacts, got %d", len(listed))
	}
	created := requireStatus(t, srv, http.MethodPost, "/contacts", map[string]any{
		"firstName": "Hector",
		"lastName":  "Ruiz",
		"email":     "hruiz@lonestarfreight.example",
	}, http.StatusOK).data(t)
	contactID := recordID(created)
	if created["phone"] != "" || created["firstName"] != "Hector" {
		t.Fatalf("expected every contact field rendered, got %v", created)
	}
	callAPI(t, srv, http.MethodPost, "/contacts", map[string]any{"email": "not-an-email"}).
		expectError(t, http.StatusBadRequest, "email")
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/contacts",
		map[string]any{"phone": strings.Repeat("1", 256)},
	).
		expectError(t, http.StatusBadRequest, "at most 255")

	updated := requireStatus(t, srv, http.MethodPatch, "/contacts/"+contactID, map[string]any{
		"phone": "512-555-0170",
	}, http.StatusOK).data(t)
	if updated["phone"] != "512-555-0170" || updated["lastName"] != "Ruiz" {
		t.Fatalf("expected a partial update, got %v", updated)
	}
	requireStatus(t, srv, http.MethodPatch, "/addresses/"+fixtureAustinYard, map[string]any{
		"contactIds": []any{"22407", contactID},
	}, http.StatusOK)

	requireStatus(t, srv, http.MethodDelete, "/contacts/"+contactID, nil, http.StatusNoContent)
	requireStatus(t, srv, http.MethodGet, "/contacts/"+contactID, nil, http.StatusNotFound)
	requireStatus(t, srv, http.MethodPatch, "/contacts/"+contactID, map[string]any{"phone": "1"},
		http.StatusNotFound)
	yard := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/addresses/"+fixtureAustinYard,
		nil,
		http.StatusOK,
	).data(t)
	if contacts := listOf(
		yard["contacts"],
	); len(contacts) != 1 ||
		mapOf(contacts[0])["id"] != "22407" {
		t.Fatalf("expected the deleted contact to leave the address, got %v", yard["contacts"])
	}
}

func recordsOf(items []any) []Record {
	out := make([]Record, 0, len(items))
	for _, item := range items {
		if mapped, ok := anyAsMap(item); ok {
			out = append(out, Record(mapped))
		}
	}
	return out
}

func TestAddressPolygonKeepsLocationInsideTheFence(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	created := requireStatus(t, srv, http.MethodPost, "/addresses", map[string]any{
		"name":             "Downtown Austin Dock",
		"formattedAddress": "500 E 7th St, Austin, TX 78701",
		"geofence":         map[string]any{"polygon": squarePolygon(30.268, -97.7375, 0.002)},
	}, http.StatusOK).data(t)
	latitude, _ := created["latitude"].(float64)
	longitude, _ := created["longitude"].(float64)
	if haversineMeters(latitude, longitude, 30.268, -97.7375) > 5 {
		t.Fatalf("expected a location inside the polygon, got %v,%v", latitude, longitude)
	}

	explicit := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/addresses/"+recordID(created),
		map[string]any{
			"latitude":  30.2685,
			"longitude": -97.737,
		},
		http.StatusOK,
	).data(t)
	if explicit["latitude"] != 30.2685 || explicit["longitude"] != -97.737 {
		t.Fatalf("expected explicit coordinates to win, got %v", explicit)
	}
}
