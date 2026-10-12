package sim

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	interchangeTemplateID  = "c77994b5-55e8-42f7-8fd7-d4afcdb4ded6"
	incidentTemplateID     = "5d21c4b8-6a97-4e30-9f1d-83b5a2c7e604"
	fieldInterchangeAsset  = "70d82dff-53dc-4b72-8b6e-a9f7835ed4a7"
	fieldInterchangeTime   = "3131091b-2b9e-45f6-882e-bb079ada6ee8"
	fieldInterchangeFence  = "1bacde37-4c8e-45e0-8a71-0556585988c5"
	fieldInterchangeType   = "0f031ca3-adfd-486a-97ae-bafcc1f24820"
	optionInterchangeDrop  = "e1c6eedc-5191-4352-842b-920415cb923f"
	fieldInterchangeSeal   = "62fd9868-7452-4b19-8ed0-cc04ac58b36a"
	fieldInterchangePhotos = "650e5520-bce9-457d-9801-3d0a22c9b9ef"
	fieldInterchangeTable  = "0a031537-fad8-44b3-8b2a-4a4a8f9f903d"
	columnTirePosition     = "cd18889d-a97e-4fab-bbb0-3db859a625bd"
	columnTreadDepth       = "fecba7e1-d4e1-4316-bbea-46610acc5317"
	columnTireCondition    = "53b54503-66c2-4067-9452-b31141d28acf"
	optionTireGood         = "f2e8a0b2-a082-47e3-80f6-39db5b4f3893"
	fieldInterchangePerson = "d3e189b1-2e0c-4a2b-ab38-603d8bc926f5"
	fieldInterchangeNotes  = "118be3e5-ca0e-4dd8-b69a-5d289a24ddb2"
	fixtureFormSubmission  = "3f9a6c2e-7b41-4d8a-9e15-c2b7d04f8a61"
	fixtureAPIUser         = "524871"
)

var pngPixel = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
}

func interchangeRevision(t *testing.T, srv *Server) string {
	t.Helper()
	templates := requireStatus(
		t, srv, http.MethodGet, "/form-templates?ids="+interchangeTemplateID, nil, http.StatusOK,
	).list(t)
	if len(templates) != 1 {
		t.Fatalf("expected the interchange template, got %d", len(templates))
	}
	return stringValue(templates[0], "revisionId")
}

func firstFixtureStopID(t *testing.T, srv *Server) string {
	t.Helper()
	route, ok := srv.live.RouteByID(srv.simNow(), fixtureRouteID)
	if !ok {
		t.Fatal("expected fixture route")
	}
	stops := routeRefFromRecord(route).Stops
	if len(stops) == 0 {
		t.Fatal("expected fixture route stops")
	}
	return stops[0].ID
}

func fullInterchangeBody(t *testing.T, srv *Server) map[string]any {
	t.Helper()
	return map[string]any{
		"formTemplate": map[string]any{"id": interchangeTemplateID, "revisionId": interchangeRevision(t, srv)},
		"status":       "notStarted",
		"title":        "Drop Trailer 2042 at Austin",
		"assignedTo":   map[string]any{"id": fixtureDriverAlex, "type": "driver"},
		"dueAtTime":    "2026-03-04T20:00:00Z",
		"routeStopId":  firstFixtureStopID(t, srv),
		"fields": []any{
			map[string]any{"id": fieldInterchangeAsset, "type": "asset",
				"assetValue": map[string]any{"asset": map[string]any{"id": fixtureTrailer2042}}},
			map[string]any{"id": fieldInterchangeTime, "type": "datetime",
				"dateTimeValue": map[string]any{"value": "2026-03-04T14:30:00-05:00"}},
			map[string]any{"id": fieldInterchangeFence, "type": "geofence",
				"geofenceValue": map[string]any{"geofence": map[string]any{"id": fixtureAustinYard}}},
			map[string]any{"id": fieldInterchangeType, "type": "multiple_choice",
				"multipleChoiceValue": map[string]any{"valueId": optionInterchangeDrop}},
			map[string]any{"id": fieldInterchangeSeal, "type": "barcode",
				"barcodeValue": map[string]any{"barcodes": []any{map[string]any{"value": "SL-104233"}}}},
			map[string]any{"id": fieldInterchangePhotos, "type": "media",
				"mediaValue": map[string]any{"mediaList": []any{map[string]any{
					"mediaType":     "image/png",
					"base64Payload": base64.StdEncoding.EncodeToString(pngPixel),
				}}}},
			map[string]any{"id": fieldInterchangeTable, "type": "table",
				"tableValue": map[string]any{"rows": []any{map[string]any{
					"id": "ee62df83-16e8-46ae-94d6-4933848f5e66",
					"cells": []any{
						map[string]any{"id": columnTirePosition, "type": "text",
							"textValue": map[string]any{"value": "Axle 1 Left"}},
						map[string]any{"id": columnTreadDepth, "type": "number",
							"numberValue": map[string]any{"value": float64(11)}},
						map[string]any{"id": columnTireCondition, "type": "multiple_choice",
							"multipleChoiceValue": map[string]any{"valueId": optionTireGood}},
					},
				}}}},
			map[string]any{"id": fieldInterchangePerson, "type": "person",
				"personValue": map[string]any{"person": map[string]any{"polymorphicUserId": "user-" + fixtureAPIUser}}},
			map[string]any{"id": fieldInterchangeNotes, "type": "text",
				"textValue": map[string]any{"value": "No damage observed"}},
		},
	}
}

func formFieldByID(t *testing.T, record Record, id string) Record {
	t.Helper()
	for _, raw := range listOf(record["fields"]) {
		if field, ok := anyAsMap(raw); ok && stringOf(field["id"]) == id {
			return Record(field)
		}
	}
	t.Fatalf("expected field %s in %v", id, record["fields"])
	return nil
}

func TestFormTemplateListFiltersAndValidates(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	all := requireStatus(t, srv, http.MethodGet, "/form-templates", nil, http.StatusOK)
	templates := all.list(t)
	if len(templates) != 6 {
		t.Fatalf("expected six fixture templates, got %d", len(templates))
	}
	if all.pagination(t)["hasNextPage"] != false {
		t.Fatalf("expected a single page, got %v", all.pagination(t))
	}
	incident := requireStatus(
		t, srv, http.MethodGet, "/form-templates?ids="+incidentTemplateID, nil, http.StatusOK,
	).list(t)
	if len(incident) != 1 || nestedString(incident[0], "approvalConfig", "type") != "singleApproval" {
		t.Fatalf("expected the incident template with a single-approval config, got %v", incident)
	}
	ids := make([]string, 101)
	for idx := range ids {
		ids[idx] = deterministicUUID("template", string(rune('a'+idx%26)), time.Duration(idx).String())
	}
	callAPI(t, srv, http.MethodGet, "/form-templates?ids="+strings.Join(ids, ","), nil).
		expectError(t, http.StatusBadRequest, "at most 100")
}

func TestFormSubmissionListRequiresIDsAndHonoursInclude(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	callAPI(t, srv, http.MethodGet, "/form-submissions", nil).expectError(t, http.StatusBadRequest, "ids")
	callAPI(t, srv, http.MethodGet, "/form-submissions?ids="+fixtureFormSubmission+"&include=tags", nil).
		expectError(t, http.StatusBadRequest, "include")

	plain := requireStatus(
		t, srv, http.MethodGet, "/form-submissions?ids="+fixtureFormSubmission, nil, http.StatusOK,
	).list(t)
	if len(plain) != 1 {
		t.Fatalf("expected the fixture submission, got %d", len(plain))
	}
	if _, present := plain[0]["externalIds"]; present {
		t.Fatalf("expected externalIds omitted without include, got %v", plain[0])
	}
	for _, key := range []string{"submittedAtTime", "submittedBy", "createdAtTime", "updatedAtTime", "isRequired"} {
		if _, ok := plain[0][key]; !ok {
			t.Fatalf("expected required field %s, got %v", key, plain[0])
		}
	}
	included := requireStatus(
		t, srv, http.MethodGet,
		"/form-submissions?include=externalIds&ids="+fixtureFormSubmission, nil, http.StatusOK,
	).list(t)
	if _, ok := included[0]["externalIds"].(map[string]any); !ok {
		t.Fatalf("expected externalIds object with include, got %v", included[0])
	}

	now := srv.simNow()
	generated := srv.live.GeneratedFormSubmissions(now, now.Add(-10*24*time.Hour), now, nil, nil)
	if len(generated) == 0 {
		t.Fatal("expected generated submissions")
	}
	oldest := recordID(generated[0])
	byID := requireStatus(
		t, srv, http.MethodGet,
		"/form-submissions?ids="+oldest+","+fixtureFormSubmission+",unknown-id", nil, http.StatusOK,
	).list(t)
	if len(byID) != 2 {
		t.Fatalf("expected the generated and fixture submissions, got %d", len(byID))
	}
}

func TestFormSubmissionCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	valid := fullInterchangeBody(t, srv)

	withField := func(field map[string]any) map[string]any {
		body := cloneRecord(valid)
		body["fields"] = []any{field}
		return body
	}
	with := func(key string, value any) map[string]any {
		body := cloneRecord(valid)
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
		{"missing template", with("formTemplate", nil), "formTemplate is required"},
		{"unknown template", with("formTemplate", map[string]any{"id": "9814a1fa-f0c6-408b-bf85-51dc3bc71ac7"}), "form template"},
		{"revision mismatch", with("formTemplate", map[string]any{"id": interchangeTemplateID, "revisionId": "1214a1fa-f0c6-408b-bf85-51dc3bc71ac7"}), "revision"},
		{"missing status", with("status", nil), "status is required"},
		{"completed status", with("status", "completed"), "notStarted"},
		{"long title", with("title", strings.Repeat("x", 256)), "255"},
		{"unknown driver", with("assignedTo", map[string]any{"id": "999", "type": "driver"}), "driver"},
		{"unknown user", with("assignedTo", map[string]any{"id": "999", "type": "user"}), "user"},
		{"assignee type", with("assignedTo", map[string]any{"id": fixtureDriverAlex, "type": "vehicle"}), "assignedTo.type"},
		{"bad due", with("dueAtTime", "tomorrow"), "dueAtTime"},
		{"unknown stop", with("routeStopId", "1"), "route stop"},
		{"external stop", with("routeStopId", "tms:1"), "route stop"},
		{"unknown field", withField(map[string]any{"id": "9814a1fa-f0c6-408b-bf85-51dc3bc71ac7", "type": "text", "textValue": map[string]any{"value": "x"}}), "is not a field"},
		{"type mismatch", withField(map[string]any{"id": fieldInterchangeNotes, "type": "number", "numberValue": map[string]any{"value": 1.0}}), "text"},
		{"signature input", withField(map[string]any{"id": fieldInterchangeNotes, "type": "signature"}), "type must be one of"},
		{"wrong value key", withField(map[string]any{"id": fieldInterchangeNotes, "type": "text", "numberValue": map[string]any{"value": 1.0}}), "only valid for number"},
		{"missing value", withField(map[string]any{"id": fieldInterchangeNotes, "type": "text"}), "textValue is required"},
		{"bad option", withField(map[string]any{"id": fieldInterchangeType, "type": "multiple_choice", "multipleChoiceValue": map[string]any{"valueId": optionTireGood}}), "not an option"},
		{"bad datetime", withField(map[string]any{"id": fieldInterchangeTime, "type": "datetime", "dateTimeValue": map[string]any{"value": "noon"}}), "RFC 3339"},
		{"vehicle for trailer field", withField(map[string]any{"id": fieldInterchangeAsset, "type": "asset", "assetValue": map[string]any{"asset": map[string]any{"id": fixtureTruck1001}}}), "accepts trailer"},
		{"unknown asset", withField(map[string]any{"id": fieldInterchangeAsset, "type": "asset", "assetValue": map[string]any{"asset": map[string]any{"id": "1"}}}), "asset"},
		{"unknown geofence", withField(map[string]any{"id": fieldInterchangeFence, "type": "geofence", "geofenceValue": map[string]any{"geofence": map[string]any{"id": "1"}}}), "address"},
		{"person format", withField(map[string]any{"id": fieldInterchangePerson, "type": "person", "personValue": map[string]any{"person": map[string]any{"polymorphicUserId": fixtureDriverAlex}}}), "driver-<driverId>"},
		{"unknown person", withField(map[string]any{"id": fieldInterchangePerson, "type": "person", "personValue": map[string]any{"person": map[string]any{"polymorphicUserId": "driver-1"}}}), "driver"},
		{"bad base64", withField(map[string]any{"id": fieldInterchangePhotos, "type": "media", "mediaValue": map[string]any{"mediaList": []any{map[string]any{"mediaType": "image/png", "base64Payload": "@@@"}}}}), "base64"},
		{"media mismatch", withField(map[string]any{"id": fieldInterchangePhotos, "type": "media", "mediaValue": map[string]any{"mediaList": []any{map[string]any{"mediaType": "image/jpeg", "base64Payload": base64.StdEncoding.EncodeToString(pngPixel)}}}}), "image/jpeg"},
		{"media type", withField(map[string]any{"id": fieldInterchangePhotos, "type": "media", "mediaValue": map[string]any{"mediaList": []any{map[string]any{"mediaType": "text/plain", "base64Payload": "aGk="}}}}), "mediaType"},
		{"empty barcode", withField(map[string]any{"id": fieldInterchangeSeal, "type": "barcode", "barcodeValue": map[string]any{"barcodes": []any{map[string]any{"value": " "}}}}), "value"},
		{"bad table column", withField(map[string]any{"id": fieldInterchangeTable, "type": "table", "tableValue": map[string]any{"rows": []any{map[string]any{"id": "ee62df83-16e8-46ae-94d6-4933848f5e66", "cells": []any{map[string]any{"id": optionTireGood, "type": "text", "textValue": map[string]any{"value": "x"}}}}}}}), "not a column"},
		{"bad table row id", withField(map[string]any{"id": fieldInterchangeTable, "type": "table", "tableValue": map[string]any{"rows": []any{map[string]any{"id": "row-1", "cells": []any{}}}}}), "UUID"},
		{"table decimals", withField(map[string]any{"id": fieldInterchangeTable, "type": "table", "tableValue": map[string]any{"rows": []any{map[string]any{"id": "ee62df83-16e8-46ae-94d6-4933848f5e66", "cells": []any{map[string]any{"id": columnTreadDepth, "type": "number", "numberValue": map[string]any{"value": 10.5}}}}}}}), "decimal"},
		{"duplicate field", with("fields", []any{
			map[string]any{"id": fieldInterchangeNotes, "type": "text", "textValue": map[string]any{"value": "a"}},
			map[string]any{"id": fieldInterchangeNotes, "type": "text", "textValue": map[string]any{"value": "b"}},
		}), "more than once"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/form-submissions", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
}

func TestFormSubmissionCreateRendersEveryFieldType(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	body := fullInterchangeBody(t, srv)

	created := requireStatus(t, srv, http.MethodPost, "/form-submissions", body, http.StatusOK).data(t)
	if !isUUID(recordID(created)) || stringValue(created, "status") != "notStarted" {
		t.Fatalf("expected a notStarted submission with a UUID, got %v", created)
	}
	if created["isRequired"] != true || nestedString(created, "assignedTo", "id") != fixtureDriverAlex {
		t.Fatalf("expected an assigned, required submission, got %v", created)
	}
	if stringValue(created, "routeId") != fixtureRouteID || stringValue(created, "routeStopId") != body["routeStopId"] {
		t.Fatalf("expected route and stop from the stop ID, got %v", created)
	}
	if nestedString(created, "submittedBy", "type") != "user" || stringValue(created, "submittedAtTime") == "" {
		t.Fatalf("expected API user as submittedBy, got %v", created)
	}
	if got := nestedString(formFieldByID(t, created, fieldInterchangeTime), "dateTimeValue", "value"); got != "2026-03-04T19:30:00Z" {
		t.Fatalf("expected UTC datetime value, got %q", got)
	}
	if got := nestedString(formFieldByID(t, created, fieldInterchangeAsset), "assetValue", "asset", "entryType"); got != "tracked" {
		t.Fatalf("expected tracked asset, got %q", got)
	}
	if got := nestedString(formFieldByID(t, created, fieldInterchangeFence), "geofenceValue", "geofence", "address"); got == "" {
		t.Fatal("expected the geofence address")
	}
	if got := nestedString(formFieldByID(t, created, fieldInterchangeType), "multipleChoiceValue", "value"); got != "Drop" {
		t.Fatalf("expected Drop option label, got %q", got)
	}
	media := listOf(mapOf(formFieldByID(t, created, fieldInterchangePhotos)["mediaValue"])["mediaList"])
	if len(media) != 1 || stringOf(mapOf(media[0])["urlExpiresAt"]) == "" {
		t.Fatalf("expected one finished media record with an expiring URL, got %v", media)
	}
	table := mapOf(formFieldByID(t, created, fieldInterchangeTable)["tableValue"])
	if len(listOf(table["columns"])) != 3 || len(listOf(table["rows"])) != 1 {
		t.Fatalf("expected the table columns and row, got %v", table)
	}
	if got := nestedString(formFieldByID(t, created, fieldInterchangePerson), "personValue", "person", "polymorphicUserId", "type"); got != "user" {
		t.Fatalf("expected the person polymorphic user, got %q", got)
	}

	unassigned := cloneRecord(body)
	delete(unassigned, "assignedTo")
	delete(unassigned, "fields")
	second := requireStatus(t, srv, http.MethodPost, "/form-submissions", unassigned, http.StatusOK).data(t)
	if second["isRequired"] != false {
		t.Fatalf("expected isRequired to default false without an assignee, got %v", second["isRequired"])
	}

	fetched := requireStatus(
		t, srv, http.MethodGet, "/form-submissions?ids="+recordID(created), nil, http.StatusOK,
	).list(t)
	if len(fetched) != 1 || len(listOf(fetched[0]["fields"])) != 9 {
		t.Fatalf("expected the stored submission with nine fields, got %v", fetched)
	}
}

func TestFormSubmissionPatchLifecycleAndWebhook(t *testing.T) {
	sink := newWebhookSink(t)
	srv := newFleetTestServer(t, fleetServerOptions{WebhookURL: sink.url})
	body := fullInterchangeBody(t, srv)
	delete(body, "fields")
	created := requireStatus(t, srv, http.MethodPost, "/form-submissions", body, http.StatusOK).data(t)
	id := recordID(created)

	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"status": "inProgress"}).
		expectError(t, http.StatusBadRequest, "id is required")
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": "missing"}).
		expectError(t, http.StatusNotFound, "")
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "approved"}).
		expectError(t, http.StatusBadRequest, "require approvals")
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "completed"}).
		expectError(t, http.StatusBadRequest, "status must be one of")
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{
		"id": id, "approvalDetails": map[string]any{"comment": "x"},
	}).expectError(t, http.StatusBadRequest, "approvalDetails")

	srv.clock.SetTime(srv.simNow().Add(10 * time.Minute))
	updated := requireStatus(t, srv, http.MethodPatch, "/form-submissions", map[string]any{
		"id":     id,
		"status": "inProgress",
		"title":  "Drop Trailer 2042 - in progress",
		"fields": []any{map[string]any{"id": fieldInterchangeNotes, "type": "text",
			"textValue": map[string]any{"value": "Left rear marker light out"}}},
	}, http.StatusOK).data(t)
	if stringValue(updated, "status") != "inProgress" || len(listOf(updated["fields"])) != 1 {
		t.Fatalf("expected inProgress with one field, got %v", updated)
	}
	if stringValue(updated, "updatedAtTime") == stringValue(created, "updatedAtTime") ||
		stringValue(updated, "createdAtTime") != stringValue(created, "createdAtTime") {
		t.Fatalf("expected updatedAtTime to move and createdAtTime to stay, got %v", updated)
	}
	events := waitForWebhookEvents(t, sink, "FormUpdated", 1)
	form := mapOf(webhookData(t, events[0])["form"])
	if stringOf(form["id"]) != id || stringOf(form["status"]) != "inProgress" {
		t.Fatalf("expected FormUpdated with the patched form, got %v", form)
	}

	requireStatus(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "archived"}, http.StatusOK)
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "inProgress"}).
		expectError(t, http.StatusBadRequest, "archived")
}

func TestFormSubmissionApprovalOnGeneratedIncidentReport(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	now := srv.simNow()
	var incident Record
	for _, record := range srv.live.GeneratedFormSubmissions(now, now.Add(-14*24*time.Hour), now, nil, nil) {
		if nestedString(record, "formTemplate", "id") == incidentTemplateID {
			incident = record
		}
	}
	if incident == nil {
		t.Fatal("expected a generated incident report")
	}
	if stringValue(incident, "status") != "needsReview" {
		t.Fatalf("expected approval-gated incident reports in needsReview, got %q", stringValue(incident, "status"))
	}
	id := recordID(incident)
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "denied"}).
		expectError(t, http.StatusBadRequest, "comment is required")
	srv.clock.SetTime(now.Add(5 * time.Minute))
	approved := requireStatus(t, srv, http.MethodPatch, "/form-submissions", map[string]any{
		"id": id, "status": "changesRequested",
		"approvalDetails": map[string]any{"comment": "Add a photo of the damage."},
	}, http.StatusOK).data(t)
	if nestedString(approved, "approvalDetails", "comment") != "Add a photo of the damage." {
		t.Fatalf("expected approval comment, got %v", approved)
	}
	callAPI(t, srv, http.MethodPatch, "/form-submissions", map[string]any{"id": id, "status": "approved"}).
		expectError(t, http.StatusBadRequest, "needsReview")

	start := url.QueryEscape(now.Add(time.Minute).Format(time.RFC3339))
	stream := requireStatus(t, srv, http.MethodGet, "/form-submissions/stream?startTime="+start, nil, http.StatusOK).list(t)
	found := 0
	for _, record := range stream {
		if recordID(record) == id {
			found++
			if stringValue(record, "status") != "changesRequested" {
				t.Fatalf("expected the patched status in the stream, got %v", record)
			}
		}
	}
	if found != 1 {
		t.Fatalf("expected the patched submission once in the stream, got %d", found)
	}
}

func TestFormSubmissionStreamFiltersAndPolling(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	now := srv.simNow()
	start := url.QueryEscape(now.Add(-24 * time.Hour).Format(time.RFC3339))
	base := "/form-submissions/stream?startTime=" + start

	for _, name := range []string{"formTemplateIds", "userIds", "driverIds", "assignedToRouteStopIds"} {
		ids := make([]string, 51)
		for idx := range ids {
			ids[idx] = "x" + time.Duration(idx).String()
		}
		callAPI(t, srv, http.MethodGet, base+"&"+name+"="+strings.Join(ids, ","), nil).
			expectError(t, http.StatusBadRequest, "at most 50")
	}
	callAPI(t, srv, http.MethodGet, base+"&after=bogus", nil).expectError(t, http.StatusBadRequest, "after")

	first := requireStatus(t, srv, http.MethodGet, base, nil, http.StatusOK)
	records := first.list(t)
	if len(records) == 0 {
		t.Fatal("expected submissions in the trailing day")
	}
	for idx := 1; idx < len(records); idx++ {
		if stringValue(records[idx-1], "updatedAtTime") > stringValue(records[idx], "updatedAtTime") {
			t.Fatal("expected the stream ordered by updatedAtTime")
		}
	}
	cursor := stringOf(first.pagination(t)["endCursor"])
	if cursor == "" {
		t.Fatal("expected a resumable cursor when polling without endTime")
	}
	caughtUp := requireStatus(t, srv, http.MethodGet, base+"&after="+cursor, nil, http.StatusOK)
	if len(caughtUp.list(t)) != 0 || stringOf(caughtUp.pagination(t)["endCursor"]) != cursor {
		t.Fatalf("expected an empty page that keeps the cursor, got %s", caughtUp.Body)
	}
	srv.clock.SetTime(now.Add(30 * time.Hour))
	later := requireStatus(t, srv, http.MethodGet, base+"&after="+cursor, nil, http.StatusOK).list(t)
	if len(later) == 0 {
		t.Fatal("expected new submissions after the clock advanced")
	}
	for _, record := range later {
		if stringValue(record, "updatedAtTime") <= stringValue(records[len(records)-1], "updatedAtTime") {
			t.Fatalf("expected only newer submissions after the cursor, got %v", record)
		}
	}

	userOnly := requireStatus(t, srv, http.MethodGet, base+"&userIds="+fixtureAPIUser, nil, http.StatusOK).list(t)
	for _, record := range userOnly {
		if nestedString(record, "submittedBy", "type") != "user" {
			t.Fatalf("expected only user submissions, got %v", record["submittedBy"])
		}
	}
	stopID := firstFixtureStopID(t, srv)
	byStop := requireStatus(t, srv, http.MethodGet, base+"&assignedToRouteStopIds="+stopID, nil, http.StatusOK).list(t)
	if len(byStop) == 0 {
		t.Fatal("expected BOL submissions on the first stop")
	}
	for _, record := range byStop {
		if stringValue(record, "routeStopId") != stopID {
			t.Fatalf("expected stop %s, got %v", stopID, record["routeStopId"])
		}
	}
	included := requireStatus(t, srv, http.MethodGet, base+"&include=externalIds", nil, http.StatusOK).list(t)
	if _, ok := included[0]["externalIds"].(map[string]any); !ok {
		t.Fatalf("expected externalIds with include, got %v", included[0])
	}
}

func TestFormPDFExportLifecycle(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	callAPI(t, srv, http.MethodPost, "/form-submissions/pdf-exports", nil).
		expectError(t, http.StatusBadRequest, "id")
	callAPI(t, srv, http.MethodPost, "/form-submissions/pdf-exports?id=missing", nil).
		expectError(t, http.StatusNotFound, "")
	callAPI(t, srv, http.MethodGet, "/form-submissions/pdf-exports", nil).
		expectError(t, http.StatusBadRequest, "pdfId")

	created := requireStatus(
		t, srv, http.MethodPost, "/form-submissions/pdf-exports?id="+fixtureFormSubmission, nil, http.StatusAccepted,
	).data(t)
	if stringValue(created, "id") != fixtureFormSubmission || stringValue(created, "jobStatus") != "pending" ||
		stringValue(created, "expiresAtTime") == "" {
		t.Fatalf("expected a pending export job, got %v", created)
	}
	pdfID := stringValue(created, "pdfId")
	pending := requireStatus(t, srv, http.MethodGet, "/form-submissions/pdf-exports?pdfId="+pdfID, nil, http.StatusOK).data(t)
	if stringValue(pending, "jobStatus") != "pending" {
		t.Fatalf("expected pending, got %v", pending)
	}
	srv.clock.SetTime(srv.simNow().Add(formPDFReadyDelay))
	done := requireStatus(t, srv, http.MethodGet, "/form-submissions/pdf-exports?pdfId="+pdfID, nil, http.StatusOK).data(t)
	if stringValue(done, "jobStatus") != "done" || stringValue(done, "pdfUrl") == "" ||
		stringValue(done, "completedAtTime") == "" || stringValue(done, "pdfUrlExpiresAtTime") == "" {
		t.Fatalf("expected a finished export with a URL, got %v", done)
	}
	srv.clock.SetTime(srv.simNow().Add(formPDFJobTTL))
	callAPI(t, srv, http.MethodGet, "/form-submissions/pdf-exports?pdfId="+pdfID, nil).
		expectError(t, http.StatusNotFound, "")
}

func TestGeneratedTrailerInterchangeSubmissions(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	now := srv.simNow()
	var interchange Record
	for _, record := range srv.live.GeneratedFormSubmissions(now, now.Add(-14*24*time.Hour), now, nil, nil) {
		if nestedString(record, "formTemplate", "id") == interchangeTemplateID {
			interchange = record
			break
		}
	}
	if interchange == nil {
		t.Fatal("expected generated trailer interchange receipts")
	}
	assetID := nestedString(formFieldByID(t, interchange, fieldInterchangeAsset), "assetValue", "asset", "id")
	trailer, ok := srv.fleetView().snap.assetByID[assetID]
	if !ok || assetType(trailer) != assetTypeTrailer {
		t.Fatalf("expected the coupled trailer, got %q", assetID)
	}
	if nestedString(formFieldByID(t, interchange, fieldInterchangeFence), "geofenceValue", "geofence", "id") == "" {
		t.Fatal("expected the nearest yard geofence")
	}
	rows := listOf(mapOf(formFieldByID(t, interchange, fieldInterchangeTable)["tableValue"])["rows"])
	if len(rows) != len(formTirePositions) || len(listOf(mapOf(rows[0])["cells"])) != 3 {
		t.Fatalf("expected one tire row per position with three cells, got %v", rows)
	}
	media := listOf(mapOf(formFieldByID(t, interchange, fieldInterchangePhotos)["mediaValue"])["mediaList"])
	if len(media) != formInterchangePhotoCount {
		t.Fatalf("expected %d photos, got %d", formInterchangePhotoCount, len(media))
	}
	if nestedString(formFieldByID(t, interchange, fieldInterchangePerson), "personValue", "person", "polymorphicUserId", "id") != fixtureAPIUser {
		t.Fatal("expected the yard user as the receiver")
	}
}
