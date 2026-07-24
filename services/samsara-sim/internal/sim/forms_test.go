package sim

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

const (
	testFuelTemplateID      = "aaaa1111-0000-4000-8000-000000000001"
	testIncidentTemplateID  = "bbbb2222-0000-4000-8000-000000000002"
	testChecklistTemplateID = "cccc3333-0000-4000-8000-000000000003"
)

func newFormTestFixture(driverCount int) *Fixture {
	fixture := newDvirFixture(driverCount)
	fixture.FormTemplates = []Record{
		{
			"id":            testFuelTemplateID,
			"revisionId":    "aaaa1111-0000-4000-8000-0000000000f1",
			"title":         "Fuel Receipt",
			"createdAtTime": "2026-01-05T08:00:00Z",
			"updatedAtTime": "2026-02-12T09:30:00Z",
			"createdBy":     map[string]any{"id": "user-1", "type": "user"},
			"updatedBy":     map[string]any{"id": "user-1", "type": "user"},
			"fields": []any{
				map[string]any{
					"id":               "aaaa1111-0000-4000-8000-00000000f001",
					"label":            "Gallons Purchased",
					"type":             "number",
					"isRequired":       true,
					"numDecimalPlaces": 2,
				},
				map[string]any{
					"id":               "aaaa1111-0000-4000-8000-00000000f002",
					"label":            "Total Amount",
					"type":             "number",
					"isRequired":       true,
					"numDecimalPlaces": 2,
				},
				map[string]any{
					"id":         "aaaa1111-0000-4000-8000-00000000f003",
					"label":      "Fuel Stop Location",
					"type":       "text",
					"isRequired": true,
				},
			},
			"sections": []any{},
		},
		{
			"id":            testIncidentTemplateID,
			"revisionId":    "bbbb2222-0000-4000-8000-0000000000f2",
			"title":         "Incident Report",
			"createdAtTime": "2026-01-05T08:15:00Z",
			"updatedAtTime": "2026-02-12T09:45:00Z",
			"createdBy":     map[string]any{"id": "user-1", "type": "user"},
			"updatedBy":     map[string]any{"id": "user-1", "type": "user"},
			"fields": []any{
				map[string]any{
					"id":         "bbbb2222-0000-4000-8000-00000000f001",
					"label":      "Incident Description",
					"type":       "text",
					"isRequired": true,
				},
				map[string]any{
					"id":         "bbbb2222-0000-4000-8000-00000000f002",
					"label":      "Incident Severity",
					"type":       "multiple_choice",
					"isRequired": true,
					"options": []any{
						map[string]any{"id": "opt-low", "label": "Low"},
						map[string]any{"id": "opt-medium", "label": "Medium"},
						map[string]any{"id": "opt-high", "label": "High"},
					},
				},
			},
			"sections": []any{},
		},
		{
			"id":            testChecklistTemplateID,
			"revisionId":    "cccc3333-0000-4000-8000-0000000000f3",
			"title":         "Trip Inspection Checklist",
			"createdAtTime": "2026-01-05T08:00:00Z",
			"updatedAtTime": "2026-02-12T09:30:00Z",
			"createdBy":     map[string]any{"id": "user-1", "type": "user"},
			"updatedBy":     map[string]any{"id": "user-1", "type": "user"},
			"fields": []any{
				map[string]any{
					"id":         "cccc3333-0000-4000-8000-00000000f001",
					"label":      "Exterior Inspection",
					"type":       "check_boxes",
					"isRequired": true,
					"options": []any{
						map[string]any{"id": "opt-lights", "label": "Lights OK"},
						map[string]any{"id": "opt-tires", "label": "Tires OK"},
						map[string]any{"id": "opt-mirrors", "label": "Mirrors OK"},
					},
				},
			},
			"sections": []any{},
		},
	}
	fixture.FormSubmissions = []Record{
		{
			"id":            "form-sub-1",
			"title":         "Truck 1 Daily Inspection",
			"status":        "notStarted",
			"isRequired":    true,
			"createdAtTime": "2026-03-01T12:00:00Z",
			"updatedAtTime": "2026-03-01T12:00:00Z",
			"formTemplate": map[string]any{
				"id":         testChecklistTemplateID,
				"revisionId": "cccc3333-0000-4000-8000-0000000000f3",
			},
			"fields": []any{},
		},
	}
	return fixture
}

func newFormTestServer(t *testing.T, driverCount int, webhookURL string) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Webhooks.Enabled = false

	fixture := newFormTestFixture(driverCount)
	if webhookURL != "" {
		cfg.Webhooks.Enabled = true
		cfg.Webhooks.MaxAttempts = 1
		cfg.Webhooks.InitialBackoff = 10 * time.Millisecond
		fixture.Webhooks = []Record{
			{
				"id":   "wh-form-test",
				"name": "form event sink",
				"url":  webhookURL,
				"simDelivery": map[string]any{
					"allowDuplicates":    false,
					"allowReorder":       false,
					"allowTimestampSkew": false,
					"retryJitterMs":      0,
				},
			},
		}
	}

	store := NewStore(fixture)
	scenarios, err := NewScenarioEngine("form-test-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}

	var dispatcher *Dispatcher
	if cfg.Webhooks.Enabled {
		dispatcher = NewDispatcher(cfg.Webhooks, store, nil)
		t.Cleanup(dispatcher.Shutdown)
	}
	return NewServer(&cfg, store, scenarios, dispatcher, nil)
}

func TestLiveSimulatorFormSubmissionsDeterministic(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(NewStore(newFormTestFixture(3)), "form-deterministic-seed")
	now := simulator.anchorTime.Add(72 * time.Hour)
	windowStart := now.Add(-48 * time.Hour)

	first := simulator.GeneratedFormSubmissions(now, windowStart, now, nil, nil)
	second := simulator.GeneratedFormSubmissions(now, windowStart, now, nil, nil)
	if len(first) == 0 {
		t.Fatal("expected generated form submissions")
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("expected identical form submission output for identical sim time")
	}
}

func TestLiveSimulatorFormSubmissionsCadenceAndValues(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(NewStore(newFormTestFixture(12)), "form-cadence-seed")
	now := simulator.anchorTime.Add(9 * 24 * time.Hour)
	windowStart := now.Add(-6 * 24 * time.Hour)

	records := simulator.GeneratedFormSubmissions(now, windowStart, now, nil, nil)
	if len(records) < 60 {
		t.Fatalf("expected large submission sample, got %d", len(records))
	}

	templateIDs := map[string]struct{}{
		testFuelTemplateID:      {},
		testIncidentTemplateID:  {},
		testChecklistTemplateID: {},
	}
	perDriverDay := map[string]int{}
	checklistTotal := 0
	checklistFull := 0
	for _, record := range records {
		if stringValue(record, "status") != formSubmissionStatusCompleted {
			t.Fatalf("unexpected status %q", stringValue(record, "status"))
		}
		if nestedString(record, "submittedBy", "type") != formSubmitterTypeDriver {
			t.Fatal("expected driver submitter")
		}
		templateID := nestedString(record, "formTemplate", "id")
		if _, ok := templateIDs[templateID]; !ok {
			t.Fatalf("unexpected template id %q", templateID)
		}

		submittedAt := mustParseRecordTime(t, record, "submittedAtTime")
		createdAt := mustParseRecordTime(t, record, "createdAtTime")
		if !submittedAt.After(createdAt) {
			t.Fatal("expected submittedAtTime after createdAtTime")
		}
		driverID := nestedString(record, "submittedBy", "id")
		perDriverDay[driverID+"|"+submittedAt.Format("2006-01-02")]++

		fields, ok := record["fields"].([]any)
		if !ok || len(fields) == 0 {
			t.Fatal("expected populated fields array")
		}
		for _, rawField := range fields {
			field, isMap := anyAsMap(rawField)
			if !isMap {
				t.Fatal("expected field input object")
			}
			switch stringValue(Record(field), "type") {
			case formFieldTypeNumber:
				value := floatFromAny(nestedAny(Record(field), "numberValue", "value"))
				if value <= 0 {
					t.Fatalf("expected positive number value, got %v", value)
				}
				if stringValue(Record(field), "label") == "Gallons Purchased" &&
					(value < 38 || value > 120) {
					t.Fatalf("expected gallons in [38,120], got %v", value)
				}
			case formFieldTypeText:
				if nestedString(Record(field), "textValue", "value") == "" {
					t.Fatal("expected non-empty text value")
				}
			case formFieldTypeMultipleChoice:
				if nestedString(Record(field), "multipleChoiceValue", "valueId") == "" ||
					nestedString(Record(field), "multipleChoiceValue", "value") == "" {
					t.Fatal("expected multiple choice value and valueId")
				}
			case formFieldTypeCheckBoxes:
				checklistTotal++
				values, valuesOK := nestedAny(Record(field), "checkBoxesValue", "value").([]any)
				valueIDs, idsOK := nestedAny(Record(field), "checkBoxesValue", "valueIds").([]any)
				if !valuesOK || !idsOK || len(values) != len(valueIDs) || len(values) == 0 {
					t.Fatal("expected aligned non-empty check box selections")
				}
				if len(values) == 3 {
					checklistFull++
				}
			default:
				t.Fatalf("unexpected field type %q", stringValue(Record(field), "type"))
			}
		}
	}

	for key, count := range perDriverDay {
		if count < 1 || count > 2 {
			t.Fatalf("expected 1-2 submissions per driver day, got %d for %s", count, key)
		}
	}
	if checklistTotal == 0 {
		t.Fatal("expected checklist submissions in sample")
	}
	if fullFraction := float64(checklistFull) / float64(checklistTotal); fullFraction < 0.6 {
		t.Fatalf("expected mostly passing checklists, got %.3f", fullFraction)
	}
}

func TestServerFormSubmissionStreamValidationFiltersAndPagination(t *testing.T) {
	t.Parallel()

	srv := newFormTestServer(t, 3, "")
	now := srv.simNow()
	startRaw := url.QueryEscape(now.Add(-48 * time.Hour).Format(time.RFC3339))
	endRaw := url.QueryEscape(now.Format(time.RFC3339))
	base := "/form-submissions/stream?startTime=" + startRaw + "&endTime=" + endRaw

	missingStart := performAuthorizedRequest(srv, http.MethodGet, "/form-submissions/stream")
	if missingStart.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing startTime, got %d", missingStart.Code)
	}
	if code := mustReadErrorCode(t, missingStart.Body.Bytes()); code != "TIME_RANGE_REQUIRED" {
		t.Fatalf("expected TIME_RANGE_REQUIRED code, got %q", code)
	}

	malformed := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/form-submissions/stream?startTime=not-a-time",
	)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed startTime, got %d", malformed.Code)
	}

	tooWide := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/form-submissions/stream?startTime="+url.QueryEscape(
			now.Add(-40*24*time.Hour).Format(time.RFC3339),
		),
	)
	if tooWide.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized window, got %d", tooWide.Code)
	}

	all := performAuthorizedRequest(srv, http.MethodGet, base)
	if all.Code != http.StatusOK {
		t.Fatalf("expected 200 for stream, got %d", all.Code)
	}
	allRecords := mustReadDataRecords(t, all.Body.Bytes())
	if len(allRecords) == 0 {
		t.Fatal("expected generated submissions in stream window")
	}

	byDriver := performAuthorizedRequest(srv, http.MethodGet, base+"&driverIds=drv-1")
	driverRecords := mustReadDataRecords(t, byDriver.Body.Bytes())
	if len(driverRecords) == 0 {
		t.Fatal("expected submissions for filtered driver")
	}
	for _, record := range driverRecords {
		if nestedString(record, "submittedBy", "id") != "drv-1" {
			t.Fatalf("unexpected submitter %q", nestedString(record, "submittedBy", "id"))
		}
	}

	byTemplate := performAuthorizedRequest(
		srv,
		http.MethodGet,
		base+"&formTemplateIds="+testFuelTemplateID,
	)
	templateRecords := mustReadDataRecords(t, byTemplate.Body.Bytes())
	for _, record := range templateRecords {
		if nestedString(record, "formTemplate", "id") != testFuelTemplateID {
			t.Fatalf("unexpected template %q", nestedString(record, "formTemplate", "id"))
		}
	}

	firstPage := performAuthorizedRequest(srv, http.MethodGet, base+"&limit=1")
	firstRecords, firstPagination := mustReadDailyLogPage(t, firstPage.Body.Bytes())
	if len(firstRecords) != 1 {
		t.Fatalf("expected 1 record on first page, got %d", len(firstRecords))
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
		base+"&limit=1&after="+url.QueryEscape(endCursor),
	)
	secondRecords, _ := mustReadDailyLogPage(t, secondPage.Body.Bytes())
	if len(secondRecords) != 1 {
		t.Fatalf("expected 1 record on second page, got %d", len(secondRecords))
	}
	if stringValue(secondRecords[0], "id") == stringValue(firstRecords[0], "id") {
		t.Fatal("expected second page to advance past first page")
	}
}

func TestServerFormSubmissionListIncludesFixtureAndGeneratedRecords(t *testing.T) {
	t.Parallel()

	srv := newFormTestServer(t, 3, "")
	now := srv.simNow()

	listResponse := performAuthorizedRequest(srv, http.MethodGet, "/form-submissions")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for submission list, got %d", listResponse.Code)
	}
	listRecords := mustReadDataRecords(t, listResponse.Body.Bytes())
	foundFixture := false
	for _, record := range listRecords {
		if stringValue(record, "id") == "form-sub-1" {
			foundFixture = true
		}
	}
	if !foundFixture {
		t.Fatal("expected fixture submission form-sub-1 in list response")
	}

	generated := srv.live.GeneratedFormSubmissions(now, now.Add(-24*time.Hour), now, nil, nil)
	if len(generated) == 0 {
		t.Fatal("expected generated submissions in trailing 24h")
	}
	generatedID := recordID(generated[0])

	byID := performAuthorizedRequest(srv, http.MethodGet, "/form-submissions?ids="+generatedID)
	byIDRecords := mustReadDataRecords(t, byID.Body.Bytes())
	if len(byIDRecords) != 1 || stringValue(byIDRecords[0], "id") != generatedID {
		t.Fatalf("expected generated submission %q via ids lookup, got %v", generatedID, byIDRecords)
	}
}

func TestServerDispatchFormWebhooksDeduplicated(t *testing.T) {
	capture := &webhookEventCapture{}
	webhookReceiver := httptest.NewServer(capture.handler(t))
	defer webhookReceiver.Close()

	srv := newFormTestServer(t, 1, webhookReceiver.URL)
	now := srv.simNow()
	submissions := srv.live.GeneratedFormSubmissions(now, now.Add(-48*time.Hour), now, nil, nil)
	if len(submissions) == 0 {
		t.Fatal("expected generated submissions for webhook test")
	}

	target := submissions[len(submissions)-1]
	targetAt := mustParseRecordTime(t, target, "submittedAtTime")
	at := targetAt.Add(time.Minute)
	windowStart := at.Add(-defaultAssetLookback)
	expected := 0
	for _, record := range submissions {
		submittedAt := mustParseRecordTime(t, record, "submittedAtTime")
		if submittedAt.After(windowStart) && !submittedAt.After(at) {
			expected++
		}
	}
	if expected == 0 {
		t.Fatal("expected at least one submission inside the lazy dispatch window")
	}

	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats?vehicleIds=veh-1", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")
	srv.dispatchFormEvents(request, at)
	srv.dispatchFormEvents(request, at)

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
		t.Fatalf("expected %d deduplicated FormSubmitted deliveries, got %d", expected, len(events))
	}
	sawTarget := false
	for _, event := range events {
		if event.EventType != "FormSubmitted" {
			t.Fatalf("unexpected event type %q", event.EventType)
		}
		data, ok := anyAsMap(event.Data)
		if !ok {
			t.Fatal("expected webhook data object")
		}
		form, ok := anyAsMap(data["form"])
		if !ok {
			t.Fatal("expected form payload in webhook data")
		}
		if stringValue(Record(form), "id") == recordID(target) {
			sawTarget = true
		}
	}
	if !sawTarget {
		t.Fatalf("expected FormSubmitted delivery for %q", recordID(target))
	}
}
