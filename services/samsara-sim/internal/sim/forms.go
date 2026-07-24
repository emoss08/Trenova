package sim

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	formFieldTypeNumber         = "number"
	formFieldTypeText           = "text"
	formFieldTypeMultipleChoice = "multiple_choice"
	formFieldTypeCheckBoxes     = "check_boxes"
	formFieldTypeSignature      = "signature"

	formSubmissionStatusCompleted = "completed"
	formSubmitterTypeDriver       = "driver"

	formTemplateTitleBillOfLading    = "Bill of Lading (Shipper)"
	formTemplateTitleProofOfDelivery = "Proof of Delivery (Consignee)"

	formSubmissionKindBOL = "bol"
	formSubmissionKindPOD = "pod"

	formMediaProcessingStatusFinished = "finished"
	formMediaURLPrefix                = "https://samsara-forms-submission-media-uploads.s3.us-west-2.amazonaws.com/"
	formMediaURLTTL                   = time.Hour

	formSecondSubmissionRate = 0.45
	formChecklistMissRate    = 0.12
	formListDefaultLookback  = 24 * time.Hour
	formListLookupLookback   = 7 * 24 * time.Hour
	formStreamMaxRangeDays   = 30
)

var (
	formFuelStopLocations = []string{
		"Georgetown Corridor Fuel Stop, Georgetown, TX",
		"Fort Worth Relay Point, Fort Worth, TX",
		"Humble Staging Lot, Humble, TX",
	}
	formIncidentDescriptions = []string{
		"Trailer door latch jammed during loading",
		"Minor fender scrape while docking",
		"Shifted load discovered at stop",
		"Debris strike on windshield",
	}
	formCorrectiveActions = []string{
		"Reported to dispatch and documented",
		"Secured load and resumed route",
		"Scheduled shop follow-up",
	}
	formPickupNotes = []string{
		"Loaded and secured at shipper dock",
		"Seal applied and count verified by driver",
		"On-time pickup, no exceptions noted",
	}
	formDeliveryNotes = []string{
		"Delivered in full, receiver signed",
		"Unloaded at consignee dock, no damage",
		"Delivery completed on schedule",
	}
	formReceiverNames = []string{
		"Dana Whitfield",
		"Chris Alvarado",
		"Robin Nakamura",
		"Sam Delgado",
		"Terry Okafor",
	}
)

type formGenerationContext struct {
	Now              time.Time
	GenericTemplates []Record
	BOLTemplate      Record
	PODTemplate      Record
	Roster           map[string]driverRoster
	RouteByDriver    map[string]string
	Waypoints        map[string][]routePoint
	GeometryCache    map[string]*routeGeometry
}

type formSubmissionSpec struct {
	Template     Record
	DriverID     string
	DriverName   string
	VehicleID    string
	Day          time.Time
	Kind         string
	SubmissionID string
	CreatedAt    time.Time
	SubmittedAt  time.Time
	RouteID      string
	RouteStopID  string
}

func (s *Server) handleFormSubmissionStream(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if startTime == nil {
		s.writeAPIError(writer, http.StatusBadRequest, ErrTimeRangeRequired)
		return
	}

	now := s.simNow()
	windowEnd := now
	if endTime != nil {
		windowEnd = *endTime
	}
	if windowEnd.Sub(*startTime) > formStreamMaxRangeDays*24*time.Hour {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	templateIDs := idsFromQuery(request.URL.Query(), "formTemplateIds")
	submitterIDs := append(
		idsFromQuery(request.URL.Query(), "driverIds"),
		idsFromQuery(request.URL.Query(), "userIds")...,
	)

	records := []Record{}
	if s.live != nil {
		records = s.live.GeneratedFormSubmissions(now, *startTime, windowEnd, templateIDs, submitterIDs)
	}
	seedRecords, listErr := s.store.List(ResourceFormSubmissions)
	if listErr != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, listErr)
		return
	}
	records = append(
		records,
		filterFormSubmissions(seedRecords, *startTime, windowEnd, templateIDs, submitterIDs)...,
	)
	sortFormSubmissions(records)

	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|form-submission-stream", payload)
}

func (l *LiveSimulator) GeneratedFormSubmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
	templateIDs []string,
	submitterIDs []string,
) []Record {
	now = now.UTC()
	windowStart = windowStart.UTC()
	windowEnd = windowEnd.UTC()
	if windowEnd.Before(windowStart) {
		windowStart, windowEnd = windowEnd, windowStart
	}

	templates := l.loadFormTemplates()
	if len(templates) == 0 {
		return []Record{}
	}
	roster := l.loadDriverRoster()
	ids := selectDriverIDs(nil, map[string]Record{}, roster)
	if len(ids) == 0 {
		return []Record{}
	}
	templateFilter := toStringSet(templateIDs)
	submitterFilter := toStringSet(submitterIDs)

	genericTemplates, bolTemplate, podTemplate := partitionFormTemplates(templates)

	ctx := formGenerationContext{
		Now:              now,
		GenericTemplates: genericTemplates,
		BOLTemplate:      bolTemplate,
		PODTemplate:      podTemplate,
		Roster:           roster,
		RouteByDriver:    l.routeIDByDriverMap(),
		Waypoints:        l.loadAssetWaypoints(),
		GeometryCache:    map[string]*routeGeometry{},
	}

	startDay := windowStart.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	endDay := windowEnd.Truncate(24 * time.Hour)
	dayCount := int(endDay.Sub(startDay)/(24*time.Hour)) + 1

	out := make([]Record, 0, len(ids)*dayCount)
	for _, driverID := range ids {
		if !matchesStringFilter(submitterFilter, driverID) {
			continue
		}
		for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
			for _, record := range l.driverDayFormSubmissions(&ctx, driverID, day) {
				submittedAt := stringValue(record, "submittedAtTime")
				if submittedAt <= windowStart.Format(time.RFC3339) ||
					submittedAt > windowEnd.Format(time.RFC3339) {
					continue
				}
				if !matchesStringFilter(templateFilter, nestedString(record, "formTemplate", "id")) {
					continue
				}
				out = append(out, record)
			}
		}
	}

	sortFormSubmissions(out)
	return out
}

func (l *LiveSimulator) loadFormTemplates() []Record {
	templates, err := l.store.List(ResourceFormTemplates)
	if err != nil {
		return []Record{}
	}
	sort.Slice(templates, func(i, j int) bool {
		return recordID(templates[i]) < recordID(templates[j])
	})
	return templates
}

func (l *LiveSimulator) driverDayFormSubmissions(
	ctx *formGenerationContext,
	driverID string,
	day time.Time,
) []Record {
	dayKey := day.Format("2006-01-02")
	entry := ctx.Roster[driverID]
	vehicleID := strings.TrimSpace(entry.VehicleID)
	driverName := firstNonEmpty(entry.Name, driverID)
	dayCtx := l.buildDailyEventContext(driverID, vehicleID, day)
	workSpan := dayCtx.DrivingEnd.Sub(dayCtx.DrivingStart)
	if workSpan <= 0 {
		return []Record{}
	}
	routeID := ctx.RouteByDriver[driverID]

	out := make([]Record, 0, 4)

	count := 1
	if l.hashFraction("form|count", driverID, dayKey) < formSecondSubmissionRate {
		count = 2
	}
	for idx := 0; idx < count && len(ctx.GenericTemplates) > 0; idx++ {
		indexKey := strconv.Itoa(idx)
		templateIndex := int(
			math.Floor(
				float64(len(ctx.GenericTemplates)) *
					l.hashFraction("form|template", driverID, dayKey, indexKey),
			),
		)
		if templateIndex >= len(ctx.GenericTemplates) {
			templateIndex = len(ctx.GenericTemplates) - 1
		}
		fraction := (0.1 + 0.75*l.hashFraction("form|time", driverID, dayKey, indexKey)) +
			0.05*float64(idx)
		submittedAt := dayCtx.DrivingStart.
			Add(time.Duration(fraction * float64(workSpan))).
			Truncate(time.Second)
		if record, ok := l.buildFormSubmission(ctx, formSubmissionSpec{
			Template:     ctx.GenericTemplates[templateIndex],
			DriverID:     driverID,
			DriverName:   driverName,
			VehicleID:    vehicleID,
			Day:          day,
			Kind:         indexKey,
			SubmissionID: formSubmissionID(day, driverID, strconv.Itoa(idx+1)),
			SubmittedAt:  submittedAt,
			RouteID:      routeID,
			RouteStopID:  "",
		}); ok {
			out = append(out, record)
		}
	}

	if ctx.BOLTemplate != nil {
		pickupOffset := time.Duration(
			(5 + 20*l.hashFraction("form|bol-time", driverID, dayKey)) * float64(time.Minute),
		)
		submittedAt := dayCtx.DrivingStart.Add(pickupOffset).Truncate(time.Second)
		if record, ok := l.buildFormSubmission(ctx, formSubmissionSpec{
			Template:     ctx.BOLTemplate,
			DriverID:     driverID,
			DriverName:   driverName,
			VehicleID:    vehicleID,
			Day:          day,
			Kind:         formSubmissionKindBOL,
			SubmissionID: formSubmissionID(day, driverID, formSubmissionKindBOL),
			SubmittedAt:  submittedAt,
			RouteID:      routeID,
			RouteStopID:  l.routeStopIDForKind(routeID, formSubmissionKindBOL),
		}); ok {
			out = append(out, record)
		}
	}

	if ctx.PODTemplate != nil {
		deliveryOffset := time.Duration(
			(5 + 25*l.hashFraction("form|pod-time", driverID, dayKey)) * float64(time.Minute),
		)
		submittedAt := dayCtx.DrivingEnd.Add(-deliveryOffset).Truncate(time.Second)
		if record, ok := l.buildFormSubmission(ctx, formSubmissionSpec{
			Template:     ctx.PODTemplate,
			DriverID:     driverID,
			DriverName:   driverName,
			VehicleID:    vehicleID,
			Day:          day,
			Kind:         formSubmissionKindPOD,
			SubmissionID: formSubmissionID(day, driverID, formSubmissionKindPOD),
			SubmittedAt:  submittedAt,
			RouteID:      routeID,
			RouteStopID:  l.routeStopIDForKind(routeID, formSubmissionKindPOD),
		}); ok {
			out = append(out, record)
		}
	}

	return out
}

func (l *LiveSimulator) buildFormSubmission(
	ctx *formGenerationContext,
	spec formSubmissionSpec,
) (Record, bool) {
	if spec.SubmittedAt.After(ctx.Now) {
		return nil, false
	}
	createdAt := spec.CreatedAt
	if createdAt.IsZero() {
		createdAt = spec.SubmittedAt.Add(
			-time.Duration(
				(4 + 10*l.hashFraction("form|created", spec.DriverID, spec.Day.Format("2006-01-02"), spec.Kind)) *
					float64(time.Minute),
			),
		).Truncate(time.Second)
	}

	record := Record{
		"id":              spec.SubmissionID,
		"title":           stringValue(spec.Template, "title") + " - " + spec.DriverName,
		"status":          formSubmissionStatusCompleted,
		"isRequired":      templateHasFieldType(spec.Template, formFieldTypeCheckBoxes),
		"createdAtTime":   createdAt.UTC().Format(time.RFC3339),
		"updatedAtTime":   spec.SubmittedAt.UTC().Format(time.RFC3339),
		"submittedAtTime": spec.SubmittedAt.UTC().Format(time.RFC3339),
		"submittedBy": map[string]any{
			"id":   spec.DriverID,
			"type": formSubmitterTypeDriver,
		},
		"formTemplate": map[string]any{
			"id":         recordID(spec.Template),
			"revisionId": stringValue(spec.Template, "revisionId"),
		},
		"externalIds": map[string]any{},
		"location":    l.formSubmissionLocation(ctx, &spec),
		"fields":      l.formFieldInputs(ctx, &spec),
	}
	if spec.RouteID != "" {
		record["routeId"] = spec.RouteID
	}
	if spec.RouteStopID != "" {
		record["routeStopId"] = spec.RouteStopID
	}
	return record, true
}

func formSubmissionID(day time.Time, driverID, suffix string) string {
	return strings.Join([]string{
		"form-sub",
		day.UTC().Format("20060102"),
		driverID,
		suffix,
	}, "-")
}

func partitionFormTemplates(templates []Record) (generic []Record, bol, pod Record) {
	generic = make([]Record, 0, len(templates))
	for _, template := range templates {
		switch stringValue(template, "title") {
		case formTemplateTitleBillOfLading:
			bol = template
		case formTemplateTitleProofOfDelivery:
			pod = template
		default:
			generic = append(generic, template)
		}
	}
	return generic, bol, pod
}

func (l *LiveSimulator) routeIDByDriverMap() map[string]string {
	routes, err := l.store.List(ResourceRoutes)
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(routes))
	for _, route := range routes {
		driverID := nestedString(route, "driver", "id")
		if driverID == "" {
			continue
		}
		if _, exists := out[driverID]; exists {
			continue
		}
		out[driverID] = recordID(route)
	}
	return out
}

func (l *LiveSimulator) routeStopIDForKind(routeID, kind string) string {
	if routeID == "" {
		return ""
	}
	stopCount := l.routeStopCount(routeID)
	if stopCount <= 0 {
		return ""
	}
	sequence := 1
	if kind == formSubmissionKindPOD {
		sequence = stopCount
	}
	return strings.Join([]string{routeID, "stop", strconv.Itoa(sequence)}, "-")
}

func (l *LiveSimulator) formSubmissionLocation(
	ctx *formGenerationContext,
	spec *formSubmissionSpec,
) map[string]any {
	geometry := l.cachedRouteGeometry(ctx.GeometryCache, ctx.Waypoints, spec.VehicleID)
	if geometry == nil || len(geometry.Points) == 0 {
		return nil
	}
	state := l.routeStateForGeometry(
		spec.VehicleID,
		geometry,
		spec.SubmittedAt,
		spec.SubmittedAt.Add(-time.Minute),
		ctx.Now,
	)
	return map[string]any{
		"latitude":  round(state.Latitude, 6),
		"longitude": round(state.Longitude, 6),
	}
}

func templateHasFieldType(template Record, fieldType string) bool {
	fields, ok := template["fields"].([]any)
	if !ok {
		return false
	}
	for _, raw := range fields {
		field, isMap := anyAsMap(raw)
		if isMap && stringValue(field, "type") == fieldType {
			return true
		}
	}
	return false
}

func (l *LiveSimulator) routeStopCount(routeID string) int {
	return clampInt(
		3+int(math.Floor(4*l.hashFraction("route-stop-count", routeID))),
		3,
		6,
	)
}

func (l *LiveSimulator) formFieldInputs(
	ctx *formGenerationContext,
	spec *formSubmissionSpec,
) []any {
	rawFields, ok := spec.Template["fields"].([]any)
	if !ok {
		return []any{}
	}

	driverID := spec.DriverID
	day := spec.Day
	dayKey := day.Format("2006-01-02")
	indexKey := spec.Kind
	out := make([]any, 0, len(rawFields))
	for _, raw := range rawFields {
		field, isMap := anyAsMap(raw)
		if !isMap {
			continue
		}
		fieldID := stringValue(field, "id")
		label := stringValue(field, "label")
		fieldType := stringValue(field, "type")
		input := map[string]any{
			"id":    fieldID,
			"label": label,
			"type":  fieldType,
		}

		valueHash := l.hashFraction("form|field", driverID, dayKey, indexKey, fieldID)
		switch fieldType {
		case formFieldTypeNumber:
			input["numberValue"] = map[string]any{
				"value": formNumberValue(label, valueHash),
			}
		case formFieldTypeText:
			input["textValue"] = map[string]any{
				"value": formTextValue(label, day, driverID, valueHash),
			}
		case formFieldTypeSignature:
			input["signatureValue"] = formSignatureValue(spec.SubmissionID, fieldID, ctx.Now)
		case formFieldTypeMultipleChoice:
			option, ok := formOptionAt(field, formChoiceIndex(field, valueHash))
			if !ok {
				continue
			}
			input["multipleChoiceValue"] = map[string]any{
				"value":   stringValue(option, "label"),
				"valueId": stringValue(option, "id"),
			}
		case formFieldTypeCheckBoxes:
			values, valueIDs := formCheckBoxSelection(
				field,
				valueHash,
				l.hashFraction("form|field-miss", driverID, dayKey, indexKey, fieldID),
			)
			input["checkBoxesValue"] = map[string]any{
				"value":    values,
				"valueIds": valueIDs,
			}
		default:
			continue
		}
		out = append(out, input)
	}
	return out
}

func formNumberValue(label string, valueHash float64) float64 {
	lowered := strings.ToLower(label)
	switch {
	case strings.Contains(lowered, "gallon"):
		return round(38+82*valueHash, 2)
	case strings.Contains(lowered, "amount"):
		return round(140+360*valueHash, 2)
	case strings.Contains(lowered, "piece"):
		return round(4+22*valueHash, 0)
	case strings.Contains(lowered, "weight"):
		return round(8000+36000*valueHash, 0)
	case strings.Contains(lowered, "temperature"):
		return round(34+4*valueHash, 1)
	default:
		return round(1+99*valueHash, 2)
	}
}

func formTextValue(label string, day time.Time, driverID string, valueHash float64) string {
	lowered := strings.ToLower(label)
	switch {
	case strings.Contains(lowered, "location"):
		return formCatalogValue(formFuelStopLocations, valueHash)
	case strings.Contains(lowered, "description"):
		return formCatalogValue(formIncidentDescriptions, valueHash)
	case strings.Contains(lowered, "action"):
		return formCatalogValue(formCorrectiveActions, valueHash)
	case strings.Contains(lowered, "receipt"):
		return fmt.Sprintf(
			"RCPT-%s-%04d",
			day.UTC().Format("20060102"),
			int(valueHash*10000)%10000,
		)
	case strings.Contains(lowered, "seal"):
		return fmt.Sprintf("SL-%06d", int(valueHash*1000000)%1000000)
	case strings.Contains(lowered, "name"):
		return formCatalogValue(formReceiverNames, valueHash)
	case strings.Contains(lowered, "note"):
		if strings.Contains(lowered, "delivery") {
			return formCatalogValue(formDeliveryNotes, valueHash)
		}
		return formCatalogValue(formPickupNotes, valueHash)
	default:
		return "No issues noted for " + driverID
	}
}

func formSignatureValue(submissionID, fieldID string, now time.Time) map[string]any {
	mediaID := deterministicEventID(submissionID, fieldID, "signature-media")
	return map[string]any{
		"media": map[string]any{
			"id":               mediaID,
			"processingStatus": formMediaProcessingStatusFinished,
			"url":              formMediaURLPrefix + mediaID,
			"urlExpiresAt":     now.Add(formMediaURLTTL).UTC().Format(time.RFC3339),
		},
	}
}

func formCatalogValue(catalog []string, valueHash float64) string {
	index := int(math.Floor(float64(len(catalog)) * valueHash))
	if index >= len(catalog) {
		index = len(catalog) - 1
	}
	return catalog[index]
}

func formChoiceIndex(field Record, valueHash float64) int {
	options, ok := field["options"].([]any)
	if !ok || len(options) == 0 {
		return 0
	}
	index := 0
	switch {
	case valueHash < 0.72:
		index = 0
	case valueHash < 0.93:
		index = 1
	default:
		index = 2
	}
	if index >= len(options) {
		index = len(options) - 1
	}
	return index
}

func formOptionAt(field Record, index int) (Record, bool) {
	options, ok := field["options"].([]any)
	if !ok || len(options) == 0 {
		return nil, false
	}
	if index < 0 || index >= len(options) {
		index = 0
	}
	option, isMap := anyAsMap(options[index])
	if !isMap {
		return nil, false
	}
	return option, true
}

func formCheckBoxSelection(
	field Record,
	valueHash float64,
	missHash float64,
) (values []any, valueIDs []any) {
	options, ok := field["options"].([]any)
	values = []any{}
	valueIDs = []any{}
	if !ok || len(options) == 0 {
		return values, valueIDs
	}

	missedIndex := -1
	if missHash < formChecklistMissRate {
		missedIndex = int(math.Floor(float64(len(options)) * valueHash))
		if missedIndex >= len(options) {
			missedIndex = len(options) - 1
		}
	}
	for idx, raw := range options {
		if idx == missedIndex {
			continue
		}
		option, isMap := anyAsMap(raw)
		if !isMap {
			continue
		}
		values = append(values, stringValue(option, "label"))
		valueIDs = append(valueIDs, stringValue(option, "id"))
	}
	return values, valueIDs
}

func filterFormSubmissions(
	records []Record,
	windowStart time.Time,
	windowEnd time.Time,
	templateIDs []string,
	submitterIDs []string,
) []Record {
	templateFilter := toStringSet(templateIDs)
	submitterFilter := toStringSet(submitterIDs)
	startRaw := windowStart.UTC().Format(time.RFC3339)
	endRaw := windowEnd.UTC().Format(time.RFC3339)

	out := make([]Record, 0, len(records))
	for _, record := range records {
		updatedAt := firstNonEmpty(
			stringValue(record, "updatedAtTime"),
			stringValue(record, "submittedAtTime"),
			stringValue(record, "createdAtTime"),
		)
		if updatedAt == "" || updatedAt <= startRaw || updatedAt > endRaw {
			continue
		}
		if !matchesStringFilter(templateFilter, nestedString(record, "formTemplate", "id")) {
			continue
		}
		if !matchesStringFilter(submitterFilter, nestedString(record, "submittedBy", "id")) {
			continue
		}
		out = append(out, cloneRecord(record))
	}
	return out
}

func sortFormSubmissions(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		left := firstNonEmpty(
			stringValue(records[i], "submittedAtTime"),
			stringValue(records[i], "updatedAtTime"),
		)
		right := firstNonEmpty(
			stringValue(records[j], "submittedAtTime"),
			stringValue(records[j], "updatedAtTime"),
		)
		if left == right {
			return recordID(records[i]) < recordID(records[j])
		}
		return left < right
	})
}

func (l *LiveSimulator) FormWebhookEmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
) []WebhookEmission {
	submissions := l.GeneratedFormSubmissions(now, windowStart, windowEnd, nil, nil)
	if len(submissions) == 0 {
		return []WebhookEmission{}
	}

	out := make([]WebhookEmission, 0, len(submissions))
	for _, record := range submissions {
		out = append(out, WebhookEmission{
			EventType: "FormSubmitted",
			UniqueKey: "FormSubmitted|" + recordID(record),
			Data:      map[string]any{"form": cloneRecord(record)},
		})
	}
	return out
}
