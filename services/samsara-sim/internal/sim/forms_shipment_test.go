package sim

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func newDefaultFixtureFormServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Webhooks.Enabled = false

	store := loadDefaultFixtureStore(t)
	scenarios, err := NewScenarioEngine("default-form-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	return NewServer(&cfg, store, scenarios, nil, nil)
}

const (
	shipperTemplateID   = "2c9b7e14-4a6d-4f81-b3e2-7d1f9a0c5b48"
	consigneeTemplateID = "d1e0b2f3-4c5a-4b6e-8f70-3a2d1c0e9b84"
)

func findTemplateByTitle(t *testing.T, templates []Record, title string) Record {
	t.Helper()
	for _, template := range templates {
		if stringValue(template, "title") == title {
			return template
		}
	}
	t.Fatalf("expected template titled %q", title)
	return nil
}

func templateFieldByLabel(t *testing.T, template Record, label string) Record {
	t.Helper()
	fields, ok := template["fields"].([]any)
	if !ok {
		t.Fatalf("expected fields on template %q", stringValue(template, "title"))
	}
	for _, raw := range fields {
		field, isMap := anyAsMap(raw)
		if isMap && stringValue(Record(field), "label") == label {
			return Record(field)
		}
	}
	t.Fatalf("expected field %q on template %q", label, stringValue(template, "title"))
	return nil
}

func TestDefaultFixtureShipperConsigneeTemplates(t *testing.T) {
	t.Parallel()

	store := loadDefaultFixtureStore(t)
	templates, err := store.List(ResourceFormTemplates)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}

	bol := findTemplateByTitle(t, templates, formTemplateTitleBillOfLading)
	if recordID(bol) != shipperTemplateID {
		t.Fatalf("expected shipper template id %q, got %q", shipperTemplateID, recordID(bol))
	}
	if stringValue(bol, "formCategory") != "routing" {
		t.Fatalf("expected routing formCategory, got %q", stringValue(bol, "formCategory"))
	}

	for label, wantType := range map[string]string{
		"Seal Number":              formFieldTypeText,
		"Pieces Loaded":            formFieldTypeNumber,
		"Gross Weight (lbs)":       formFieldTypeNumber,
		"Trailer Temperature (°F)": formFieldTypeNumber,
		"Load Secured":             formFieldTypeMultipleChoice,
		"Shipper Signature":        formFieldTypeSignature,
		"Pickup Notes":             formFieldTypeText,
	} {
		field := templateFieldByLabel(t, bol, label)
		if got := stringValue(field, "type"); got != wantType {
			t.Fatalf("field %q expected type %q, got %q", label, wantType, got)
		}
	}
	loadSecured := templateFieldByLabel(t, bol, "Load Secured")
	options, ok := loadSecured["options"].([]any)
	if !ok || len(options) != 2 {
		t.Fatalf("expected Load Secured to have 2 options, got %v", loadSecured["options"])
	}

	pod := findTemplateByTitle(t, templates, formTemplateTitleProofOfDelivery)
	if recordID(pod) != consigneeTemplateID {
		t.Fatalf("expected consignee template id %q, got %q", consigneeTemplateID, recordID(pod))
	}
	for label, wantType := range map[string]string{
		"Pieces Delivered":          formFieldTypeNumber,
		"Delivery Temperature (°F)": formFieldTypeNumber,
		"Condition on Arrival":      formFieldTypeMultipleChoice,
		"Seal Intact":               formFieldTypeMultipleChoice,
		"Receiver Name":             formFieldTypeText,
		"Receiver Signature":        formFieldTypeSignature,
		"Delivery Notes":            formFieldTypeText,
	} {
		field := templateFieldByLabel(t, pod, label)
		if got := stringValue(field, "type"); got != wantType {
			t.Fatalf("field %q expected type %q, got %q", label, wantType, got)
		}
	}
	condition := templateFieldByLabel(t, pod, "Condition on Arrival")
	conditionOptions, ok := condition["options"].([]any)
	if !ok || len(conditionOptions) != 3 {
		t.Fatalf("expected Condition on Arrival to have 3 options, got %v", condition["options"])
	}
}

func generatedShipperConsignee(t *testing.T) (bol, pod Record) {
	t.Helper()

	live := NewLiveSimulator(loadDefaultFixtureStore(t), "shipment-form-seed")
	now := live.anchorTime.Add(3 * 24 * time.Hour)
	windowStart := now.Add(-2 * 24 * time.Hour)
	records := live.GeneratedFormSubmissions(now, windowStart, now, nil, []string{"drv-1"})
	if len(records) == 0 {
		t.Fatal("expected generated submissions for drv-1")
	}

	bolByDay := map[string]Record{}
	podByDay := map[string]Record{}
	for _, record := range records {
		day := submissionDayToken(recordID(record))
		switch nestedString(record, "formTemplate", "id") {
		case shipperTemplateID:
			if _, ok := bolByDay[day]; !ok {
				bolByDay[day] = record
			}
		case consigneeTemplateID:
			if _, ok := podByDay[day]; !ok {
				podByDay[day] = record
			}
		}
	}
	for day, bolRecord := range bolByDay {
		if podRecord, ok := podByDay[day]; ok {
			return bolRecord, podRecord
		}
	}
	t.Fatal("expected a paired Bill of Lading and Proof of Delivery on the same sim day for drv-1")
	return nil, nil
}

func submissionDayToken(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) >= 3 {
		return parts[2]
	}
	return id
}

func TestBillOfLadingSubmissionShape(t *testing.T) {
	t.Parallel()

	bol, _ := generatedShipperConsignee(t)

	if nestedString(bol, "submittedBy", "id") != "drv-1" {
		t.Fatalf("expected submitter drv-1, got %q", nestedString(bol, "submittedBy", "id"))
	}
	if nestedString(bol, "submittedBy", "type") != formSubmitterTypeDriver {
		t.Fatal("expected driver submitter type")
	}
	if stringValue(bol, "routeId") != "route-1" {
		t.Fatalf("expected routeId route-1, got %q", stringValue(bol, "routeId"))
	}
	if stopID := stringValue(bol, "routeStopId"); !strings.HasSuffix(stopID, "-stop-1") {
		t.Fatalf("expected pickup routeStopId ending -stop-1, got %q", stopID)
	}
	if _, ok := bol["externalIds"].(map[string]any); !ok {
		t.Fatalf("expected externalIds object, got %T", bol["externalIds"])
	}
	location, ok := bol["location"].(map[string]any)
	if !ok {
		t.Fatalf("expected location object, got %T", bol["location"])
	}
	if floatFromAny(location["latitude"]) == 0 || floatFromAny(location["longitude"]) == 0 {
		t.Fatalf("expected non-zero location, got %v", location)
	}

	fields, ok := bol["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatal("expected populated BOL fields")
	}
	sawSignature := false
	for _, raw := range fields {
		field, isMap := anyAsMap(raw)
		if !isMap {
			t.Fatal("expected field input object")
		}
		record := Record(field)
		label := stringValue(record, "label")
		switch stringValue(record, "type") {
		case formFieldTypeNumber:
			value := floatFromAny(nestedAny(record, "numberValue", "value"))
			switch label {
			case "Pieces Loaded":
				if value < 4 || value > 26 {
					t.Fatalf("pieces out of range: %v", value)
				}
			case "Gross Weight (lbs)":
				if value < 8000 || value > 44000 {
					t.Fatalf("weight out of range: %v", value)
				}
			case "Trailer Temperature (°F)":
				if value < 34 || value > 38 {
					t.Fatalf("temperature out of range: %v", value)
				}
			}
		case formFieldTypeText:
			text := nestedString(record, "textValue", "value")
			if text == "" {
				t.Fatalf("expected text for %q", label)
			}
			if label == "Seal Number" && !strings.HasPrefix(text, "SL-") {
				t.Fatalf("expected seal number SL- prefix, got %q", text)
			}
		case formFieldTypeMultipleChoice:
			if nestedString(record, "multipleChoiceValue", "value") == "" ||
				nestedString(record, "multipleChoiceValue", "valueId") == "" {
				t.Fatalf("expected multiple choice value and valueId for %q", label)
			}
		case formFieldTypeSignature:
			sawSignature = true
			media, mok := anyAsMap(nestedAny(record, "signatureValue", "media"))
			if !mok {
				t.Fatalf("expected signature media object for %q", label)
			}
			if stringValue(Record(media), "id") == "" {
				t.Fatal("expected signature media id")
			}
			if !strings.HasPrefix(stringValue(Record(media), "url"), formMediaURLPrefix) {
				t.Fatalf("expected signature media url prefix, got %q", stringValue(Record(media), "url"))
			}
			if stringValue(Record(media), "urlExpiresAt") == "" {
				t.Fatal("expected signature media urlExpiresAt")
			}
		default:
			t.Fatalf("unexpected field type %q", stringValue(record, "type"))
		}
	}
	if !sawSignature {
		t.Fatal("expected a signature field in BOL submission")
	}
}

func TestProofOfDeliverySubmissionShape(t *testing.T) {
	t.Parallel()

	bol, pod := generatedShipperConsignee(t)

	bolSubmitted := mustParseRecordTime(t, bol, "submittedAtTime")
	podSubmitted := mustParseRecordTime(t, pod, "submittedAtTime")
	if !podSubmitted.After(bolSubmitted) {
		t.Fatalf("expected POD (%s) after BOL (%s)", podSubmitted, bolSubmitted)
	}
	if stopID := stringValue(pod, "routeStopId"); strings.HasSuffix(stopID, "-stop-1") || stopID == "" {
		t.Fatalf("expected delivery routeStopId at a later stop, got %q", stopID)
	}
	if stringValue(pod, "routeId") != "route-1" {
		t.Fatalf("expected routeId route-1, got %q", stringValue(pod, "routeId"))
	}

	fields, ok := pod["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatal("expected populated POD fields")
	}
	receiverName := ""
	for _, raw := range fields {
		field, _ := anyAsMap(raw)
		record := Record(field)
		if stringValue(record, "label") == "Receiver Name" {
			receiverName = nestedString(record, "textValue", "value")
		}
	}
	if receiverName == "" {
		t.Fatal("expected receiver name text value")
	}
}

func TestShipperConsigneeSubmissionsDeterministic(t *testing.T) {
	t.Parallel()

	first := NewLiveSimulator(loadDefaultFixtureStore(t), "shipment-det-seed")
	second := NewLiveSimulator(loadDefaultFixtureStore(t), "shipment-det-seed")
	now := first.anchorTime.Add(3 * 24 * time.Hour)
	windowStart := now.Add(-2 * 24 * time.Hour)

	firstRecords := first.GeneratedFormSubmissions(now, windowStart, now, nil, nil)
	secondRecords := second.GeneratedFormSubmissions(now, windowStart, now, nil, nil)
	if len(firstRecords) == 0 {
		t.Fatal("expected generated submissions")
	}
	if !reflect.DeepEqual(firstRecords, secondRecords) {
		t.Fatal("expected identical shipment submission output for identical sim time")
	}
}

func TestFormSubmissionStreamFilterByShipperTemplate(t *testing.T) {
	t.Parallel()

	srv := newDefaultFixtureFormServer(t)
	now := srv.simNow()
	startRaw := url.QueryEscape(now.Add(-3 * 24 * time.Hour).Format(time.RFC3339))
	endRaw := url.QueryEscape(now.Format(time.RFC3339))
	target := "/form-submissions/stream?startTime=" + startRaw + "&endTime=" + endRaw +
		"&formTemplateIds=" + shipperTemplateID

	response := performAuthorizedRequest(srv, http.MethodGet, target)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	records := mustReadDataRecords(t, response.Body.Bytes())
	if len(records) == 0 {
		t.Fatal("expected shipper submissions in stream window")
	}
	for _, record := range records {
		if nestedString(record, "formTemplate", "id") != shipperTemplateID {
			t.Fatalf("unexpected template %q", nestedString(record, "formTemplate", "id"))
		}
		if stringValue(record, "title") != formTemplateTitleBillOfLading+" - Alex Rivera" &&
			!strings.HasPrefix(stringValue(record, "title"), formTemplateTitleBillOfLading) {
			t.Fatalf("unexpected title %q", stringValue(record, "title"))
		}
	}
}
