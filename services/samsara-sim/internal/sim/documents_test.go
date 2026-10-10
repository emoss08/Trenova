package sim

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	docTypeBOL    = "0b98705f-2348-4951-ac38-8227f739c512"
	docTypePOD    = "be808532-9681-462c-bd82-acdf5849ce4f"
	docTypeFuel   = "35f78a3f-d7b6-482f-81a5-1021ac1a15be"
	docTypeLumper = "edb6b4bf-d44f-4bea-beb4-98d05b7e1811"
	docTypeScale  = "83a01efc-360f-4973-be3d-f3f54e3930ec"
)

func documentWindow(srv *Server, back time.Duration) string {
	now := srv.simNow()
	return "startTime=" + url.QueryEscape(now.Add(-back).Format(time.RFC3339)) +
		"&endTime=" + url.QueryEscape(now.Format(time.RFC3339))
}

func TestDocumentTypesFixture(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	result := requireStatus(t, srv, http.MethodGet, "/fleet/document-types", nil, http.StatusOK)
	types := result.list(t)
	if len(types) != 5 {
		t.Fatalf("expected five document types, got %d", len(types))
	}
	if result.pagination(t)["hasNextPage"] != false {
		t.Fatalf("expected one page, got %v", result.pagination(t))
	}
	pod := requireRecord(t, types, docTypePOD)
	if stringValue(pod, "name") != "Proof of Delivery" || len(listOf(pod["fieldTypes"])) != 7 {
		t.Fatalf("expected the POD type with seven fields, got %v", pod)
	}
	sections := listOf(pod["conditionalFieldSections"])
	if len(sections) != 1 || stringOf(mapOf(sections[0])["triggeringFieldValue"]) != "Damage" {
		t.Fatalf("expected the Damage conditional section, got %v", sections)
	}
	for _, docType := range types {
		if orgID, _ := int64Value(docType["orgId"]); orgID != webhookOrgID {
			t.Fatalf("expected orgId %d, got %v", webhookOrgID, docType["orgId"])
		}
	}
}

func TestDocumentListGeneratedAndFilters(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	callAPI(t, srv, http.MethodGet, "/fleet/documents", nil).expectError(t, http.StatusBadRequest, "required")
	callAPI(t, srv, http.MethodGet, "/fleet/documents?"+documentWindow(srv, time.Hour)+"&queryBy=submitted", nil).
		expectError(t, http.StatusBadRequest, "queryBy")
	callAPI(t, srv, http.MethodGet, "/fleet/documents?startTime=bad&endTime=bad", nil).
		expectError(t, http.StatusBadRequest, "startTime")

	window := documentWindow(srv, 3*24*time.Hour)
	documents := requireStatus(t, srv, http.MethodGet, "/fleet/documents?"+window, nil, http.StatusOK).list(t)
	if len(documents) == 0 {
		t.Fatal("expected generated driver documents")
	}
	stopIDs := map[string]struct{}{}
	for _, route := range srv.live.routeRefs(srv.simNow()) {
		for _, stop := range route.Stops {
			stopIDs[stop.ID] = struct{}{}
		}
	}
	types := map[string]int{}
	for idx, document := range documents {
		if idx > 0 && stringValue(documents[idx-1], "createdAtTime") > stringValue(document, "createdAtTime") {
			t.Fatal("expected documents ordered by createdAtTime")
		}
		types[nestedString(document, "documentType", "name")]++
		if stringValue(document, "state") != "submitted" || nestedString(document, "driver", "id") == "" {
			t.Fatalf("expected a submitted driver document, got %v", document)
		}
		if len(listOf(document["fields"])) == 0 {
			t.Fatalf("expected populated fields, got %v", document)
		}
		if stopID := nestedString(document, "routeStop", "id"); stopID != "" {
			if _, ok := stopIDs[stopID]; !ok {
				t.Fatalf("expected route stop %s to match /fleet/routes stops", stopID)
			}
			if nestedString(document, "route", "id") == "" {
				t.Fatal("expected the route alongside the route stop")
			}
		}
	}
	if types["Bill of Lading"] == 0 || types["Proof of Delivery"] == 0 {
		t.Fatalf("expected BOL and POD documents, got %v", types)
	}

	fuel := requireStatus(
		t, srv, http.MethodGet, "/fleet/documents?"+window+"&documentTypeId="+docTypeFuel, nil, http.StatusOK,
	).list(t)
	for _, document := range fuel {
		if nestedString(document, "documentType", "id") != docTypeFuel {
			t.Fatalf("expected only fuel receipts, got %v", document["documentType"])
		}
		var gallons, price, total float64
		for _, raw := range listOf(document["fields"]) {
			field := mapOf(raw)
			value, _ := mapOf(field["value"])["numberValue"].(float64)
			switch stringOf(field["label"]) {
			case "Gallons":
				gallons = value
			case "Price per Gallon":
				price = value
			case "Total Amount":
				total = value
			}
		}
		if total != round(gallons*price, 2) {
			t.Fatalf("expected total %.2f for %.3f gal at %.3f, got %.2f", round(gallons*price, 2), gallons, price, total)
		}
	}
	updated := requireStatus(
		t, srv, http.MethodGet, "/fleet/documents?"+window+"&queryBy=updated", nil, http.StatusOK,
	).list(t)
	if len(updated) != len(documents) {
		t.Fatalf("expected the same generated documents by updated time, got %d and %d", len(updated), len(documents))
	}
}

func TestDocumentGetAndDeleteGenerated(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	window := documentWindow(srv, 2*24*time.Hour)
	documents := requireStatus(t, srv, http.MethodGet, "/fleet/documents?"+window, nil, http.StatusOK).list(t)
	target := documents[0]
	id := recordID(target)

	fetched := requireStatus(t, srv, http.MethodGet, "/fleet/documents/"+id, nil, http.StatusOK).data(t)
	if recordID(fetched) != id || stringValue(fetched, "name") == "" {
		t.Fatalf("expected the generated document, got %v", fetched)
	}
	callAPI(t, srv, http.MethodGet, "/fleet/documents/missing", nil).expectError(t, http.StatusNotFound, "")
	callAPI(t, srv, http.MethodDelete, "/fleet/documents/missing", nil).expectError(t, http.StatusNotFound, "")

	response := callAPI(t, srv, http.MethodDelete, "/fleet/documents/"+id, nil)
	response.expect(t, http.StatusNoContent)
	callAPI(t, srv, http.MethodGet, "/fleet/documents/"+id, nil).expectError(t, http.StatusNotFound, "")
	for _, document := range requireStatus(t, srv, http.MethodGet, "/fleet/documents?"+window, nil, http.StatusOK).list(t) {
		if recordID(document) == id {
			t.Fatal("expected the deleted document to leave the listing")
		}
	}
}

func TestDocumentCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	base := map[string]any{"documentTypeId": docTypeLumper, "driverId": fixtureDriverAlex}
	with := func(key string, value any) map[string]any {
		body := cloneRecord(base)
		body[key] = value
		return body
	}
	field := func(label, kind string, value any) map[string]any {
		return with("fields", []any{map[string]any{"label": label, "type": kind, "value": value}})
	}
	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{"missing type", map[string]any{"driverId": fixtureDriverAlex}, "documentTypeId is required"},
		{"unknown type", with("documentTypeId", "9814a1fa-f0c6-408b-bf85-51dc3bc71ac7"), "document type"},
		{"missing driver", map[string]any{"documentTypeId": docTypeLumper}, "driverId is required"},
		{"unknown driver", with("driverId", "42"), "driver"},
		{"bad state", with("state", "archived"), "state"},
		{"long notes", with("notes", strings.Repeat("n", 2001)), "2000"},
		{"unknown vehicle", with("vehicleId", "tmsVehicleId:none"), "vehicle"},
		{"unknown stop", with("routeStopId", "1"), "route stop"},
		{"unknown label", field("Tip", "number", map[string]any{"numberValue": 5.0}), "not a field"},
		{"type mismatch", field("Amount", "string", map[string]any{"stringValue": "5"}), "number"},
		{"bad type enum", field("Amount", "money", map[string]any{"numberValue": 5.0}), "type must be one of"},
		{"wrong value key", field("Amount", "number", map[string]any{"stringValue": "5"}), "only present for string"},
		{"too many decimals", field("Amount", "number", map[string]any{"numberValue": 12.345}), "decimal"},
		{"unknown choice", field("Payment Method", "multipleChoice", map[string]any{"multipleChoiceValue": []any{map[string]any{"value": "Venmo", "selected": true}}}), "not an option"},
		{"two choices", field("Payment Method", "multipleChoice", map[string]any{"multipleChoiceValue": []any{map[string]any{"value": "Cash", "selected": true}, map[string]any{"value": "Comchek", "selected": true}}}), "only one"},
		{"photo shape", field("Receipt Photo", "photo", map[string]any{"photoValue": []any{map[string]any{"id": "p1"}}}), "url"},
		{"required value", field("Amount", "number", nil), "required"},
		{"submitted missing required", with("state", "submitted"), "required field"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/fleet/documents", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
}

func TestDocumentCreateGetDeleteAndPDF(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	route, ok := srv.live.RouteByID(srv.simNow(), fixtureRouteID)
	if !ok {
		t.Fatal("expected fixture route")
	}
	stop := routeRefFromRecord(route).Stops[0]
	body := map[string]any{
		"documentTypeId": docTypeLumper,
		"driverId":       "workerId:worker-1001",
		"vehicleId":      "samsara.vin:1FUJGLDR5CLBP1001",
		"routeStopId":    stop.ID,
		"name":           "Lumper - DC 14",
		"notes":          "Paid at dock door 6",
		"state":          "submitted",
		"fields": []any{
			map[string]any{"label": "Receipt Photo", "type": "photo", "value": map[string]any{"photoValue": []any{
				map[string]any{"id": "f5271458-21f9-4a9f-a290-780c6d8840ff", "url": "https://example.com/receipt.jpg"},
			}}},
			map[string]any{"label": "Lumper Service", "type": "string", "value": map[string]any{"stringValue": "Capstone Logistics"}},
			map[string]any{"label": "Amount", "type": "number", "value": map[string]any{"numberValue": 185.5}},
			map[string]any{"label": "Payment Method", "type": "multipleChoice", "value": map[string]any{
				"multipleChoiceValue": []any{map[string]any{"value": "Comchek", "selected": true}},
			}},
			map[string]any{"label": "Check Number", "type": "string", "value": map[string]any{"stringValue": "0012345678"}},
		},
	}
	created := requireStatus(t, srv, http.MethodPost, "/fleet/documents", body, http.StatusOK).data(t)
	id := recordID(created)
	if !isUUID(id) || stringValue(created, "createdAtTime") == "" || stringValue(created, "updatedAtTime") == "" {
		t.Fatalf("expected a stored document with timestamps, got %v", created)
	}
	if nestedString(created, "driver", "id") != fixtureDriverAlex || nestedString(created, "vehicle", "id") != fixtureTruck1001 {
		t.Fatalf("expected external IDs resolved to Alex and Truck 1001, got %v", created)
	}
	if nestedString(created, "routeStop", "id") != stop.ID || nestedString(created, "route", "id") != fixtureRouteID {
		t.Fatalf("expected the route stop and its route, got %v", created)
	}
	choices := listOf(mapOf(mapOf(listOf(created["fields"])[3])["value"])["multipleChoiceValue"])
	if len(choices) != 4 {
		t.Fatalf("expected every payment option rendered with its selection, got %v", choices)
	}

	required := requireStatus(t, srv, http.MethodPost, "/fleet/documents", map[string]any{
		"documentTypeId": docTypeScale, "driverId": fixtureDriverAlex,
	}, http.StatusOK).data(t)
	if stringValue(required, "state") != "required" || stringValue(required, "name") != "Scale Ticket" {
		t.Fatalf("expected a required document named after its type, got %v", required)
	}

	requireStatus(t, srv, http.MethodGet, "/fleet/documents/"+id, nil, http.StatusOK)
	listed := requireStatus(
		t, srv, http.MethodGet, "/fleet/documents?"+documentWindow(srv, time.Minute)+"&documentTypeId="+docTypeLumper, nil, http.StatusOK,
	).list(t)
	if len(listed) != 1 || recordID(listed[0]) != id {
		t.Fatalf("expected the created lumper receipt in the listing, got %v", listed)
	}

	callAPI(t, srv, http.MethodPost, "/fleet/documents/pdfs", map[string]any{}).
		expectError(t, http.StatusBadRequest, "documentId")
	callAPI(t, srv, http.MethodPost, "/fleet/documents/pdfs", map[string]any{"documentId": "missing"}).
		expectError(t, http.StatusNotFound, "")
	job := requireStatus(t, srv, http.MethodPost, "/fleet/documents/pdfs", map[string]any{"documentId": id}, http.StatusOK).data(t)
	if stringValue(job, "documentId") != id || !isUUID(recordID(job)) {
		t.Fatalf("expected a PDF job for the document, got %v", job)
	}
	pdfPath := "/fleet/documents/pdfs/" + recordID(job)
	status := func() Record {
		return requireStatus(t, srv, http.MethodGet, pdfPath, nil, http.StatusOK).data(t)
	}
	if got := stringValue(status(), "jobStatus"); got != "requested" {
		t.Fatalf("expected requested, got %q", got)
	}
	srv.clock.SetTime(srv.simNow().Add(documentPDFProcessing))
	if got := stringValue(status(), "jobStatus"); got != "processing" {
		t.Fatalf("expected processing, got %q", got)
	}
	srv.clock.SetTime(srv.simNow().Add(documentPDFCompleted))
	done := status()
	if stringValue(done, "jobStatus") != "completed" || stringValue(done, "downloadDocumentPdfUrl") == "" ||
		stringValue(done, "completedAtTime") == "" {
		t.Fatalf("expected a completed PDF with a URL, got %v", done)
	}
	callAPI(t, srv, http.MethodGet, "/fleet/documents/pdfs/missing", nil).expectError(t, http.StatusNotFound, "")

	callAPI(t, srv, http.MethodDelete, "/fleet/documents/"+id, nil).expect(t, http.StatusNoContent)
	callAPI(t, srv, http.MethodGet, "/fleet/documents/"+id, nil).expectError(t, http.StatusNotFound, "")
}

func TestDocumentSubmittedWebhooks(t *testing.T) {
	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{WebhookURL: sink.url})
	now := srv.simNow()
	recent := srv.live.GeneratedDocuments(now, now.Add(-48*time.Hour), now)
	if len(recent) == 0 {
		t.Fatal("expected generated documents")
	}
	at := mustRFC3339(t, stringValue(recent[len(recent)-1], "createdAtTime")).Add(time.Minute)
	generated := srv.live.GeneratedDocuments(at, at.Add(-defaultAssetLookback), at)
	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats/feed", nil)
	srv.dispatchDocumentEvents(request, at)
	srv.dispatchDocumentEvents(request, at)
	events := waitForWebhookEvents(t, sink, documentEventSubmitted, len(generated))
	time.Sleep(100 * time.Millisecond)
	if extra := waitForWebhookEvents(t, sink, documentEventSubmitted, 0); len(extra) != len(generated) {
		t.Fatalf("expected %d deduplicated DocumentSubmitted events, got %d", len(generated), len(extra))
	}
	document := mapOf(webhookData(t, events[0])["document"])
	for _, key := range []string{"id", "documentType", "driver", "fields", "state", "createdAtTime"} {
		if _, ok := document[key]; !ok {
			t.Fatalf("expected %s in the DocumentSubmitted payload, got %v", key, document)
		}
	}
}

func TestDocumentAndExportRecordsPersistAndReset(t *testing.T) {
	path := t.TempDir() + "/state.json"
	original := loadDefaultFixtureStore(t)
	enableTestPersistence(t, original, path)
	at := time.Date(2026, time.March, 4, 15, 0, 0, 0, time.UTC)
	created := map[Resource]string{}
	for _, resource := range []Resource{ResourceDocuments, ResourceDocumentPDFs, ResourceFormPDFExports} {
		record, err := original.CreateFromAPI(resource, Record{"requestedAtTime": at.Format(time.RFC3339)}, CreateOptions{At: at})
		if err != nil {
			t.Fatalf("create %s: %v", resource, err)
		}
		if !isUUID(recordID(record)) {
			t.Fatalf("expected a UUID for %s, got %q", resource, recordID(record))
		}
		created[resource] = recordID(record)
	}
	if err := original.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	restored := loadDefaultFixtureStore(t)
	enableTestPersistence(t, restored, path)
	for resource, id := range created {
		if _, err := restored.Get(resource, id); err != nil {
			t.Fatalf("expected %s %s to survive a restart: %v", resource, id, err)
		}
	}
	next, err := restored.CreateFromAPI(ResourceDocuments, Record{}, CreateOptions{At: at})
	if err != nil || recordID(next) == created[ResourceDocuments] {
		t.Fatalf("expected a fresh document ID after restart, got %v %v", next, err)
	}
	if err = restored.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	for resource, id := range created {
		if _, err = restored.Get(resource, id); err == nil {
			t.Fatalf("expected reset to clear %s %s", resource, id)
		}
	}
	types, err := restored.List(ResourceDocumentTypes)
	if err != nil || len(types) != 5 {
		t.Fatalf("expected reset to keep the five fixture document types, got %d %v", len(types), err)
	}
	if err = restored.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}
}
