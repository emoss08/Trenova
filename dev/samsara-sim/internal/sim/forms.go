package sim

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	formFieldTypeNumber         = "number"
	formFieldTypeText           = "text"
	formFieldTypeMultipleChoice = "multiple_choice"
	formFieldTypeCheckBoxes     = "check_boxes"
	formFieldTypeSignature      = "signature"

	formSubmissionStatusCompleted   = "completed"
	formSubmissionStatusNeedsReview = "needsReview"
	formSubmitterTypeDriver         = "driver"

	formTemplateTitleBillOfLading       = "Bill of Lading (Shipper)"
	formTemplateTitleProofOfDelivery    = "Proof of Delivery (Consignee)"
	formTemplateTitleTrailerInterchange = "Trailer Interchange Receipt"

	formSubmissionKindBOL         = "bol"
	formSubmissionKindPOD         = "pod"
	formSubmissionKindInterchange = "interchange"
	formInterchangeRate           = 0.22
	formInterchangePhotoCount     = 2

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
	formTirePositions = []string{
		"Axle 1 Left",
		"Axle 1 Right",
		"Axle 2 Left",
		"Axle 2 Right",
	}
)

type formGenerationContext struct {
	Now                 time.Time
	GenericTemplates    []Record
	BOLTemplate         Record
	PODTemplate         Record
	InterchangeTemplate Record
	Roster              map[string]driverRoster
	RouteByDriver       map[string]routeRef
	TrailerByVehicle    map[string]string
	Geofences           []geofenceCircle
	APIUserID           string
	Waypoints           map[string][]routePoint
	GeometryCache       map[string]*routeGeometry
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

	partition := partitionFormTemplates(templates)
	snapshot := l.fleet()

	ctx := formGenerationContext{
		Now:                 now,
		GenericTemplates:    partition.Generic,
		BOLTemplate:         partition.BOL,
		PODTemplate:         partition.POD,
		InterchangeTemplate: partition.Interchange,
		Roster:              roster,
		RouteByDriver:       routeRefsByDriver(l.routeRefs(now)),
		TrailerByVehicle:    coupledTrailersByVehicle(snapshot.assets),
		Geofences:           snapshot.geofences,
		APIUserID:           formAPIUserID(templates),
		Waypoints:           l.loadAssetWaypoints(),
		GeometryCache:       map[string]*routeGeometry{},
	}

	if windowStart.Before(telemetryEpoch) {
		windowStart = telemetryEpoch
	}
	if !windowEnd.After(windowStart) {
		return []Record{}
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
				if !matchesStringFilter(
					templateFilter,
					nestedString(record, "formTemplate", "id"),
				) {
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
	driverName := stringutils.FirstNonEmptyTrimmed(entry.Name, driverID)
	dayCtx := l.buildDailyEventContext(driverID, vehicleID, day)
	workSpan := dayCtx.DrivingEnd.Sub(dayCtx.DrivingStart)
	if workSpan <= 0 {
		return []Record{}
	}
	route := ctx.RouteByDriver[driverID]
	routeID := route.ID

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
			RouteStopID:  routeStopIDAt(&route, formSubmissionKindBOL),
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
			RouteStopID:  routeStopIDAt(&route, formSubmissionKindPOD),
		}); ok {
			out = append(out, record)
		}
	}

	if record, ok := l.interchangeSubmission(ctx, &dayCtx, driverID, driverName, vehicleID, day); ok {
		out = append(out, record)
	}

	return out
}

func (l *LiveSimulator) interchangeSubmission(
	ctx *formGenerationContext,
	dayCtx *dailyEventContext,
	driverID string,
	driverName string,
	vehicleID string,
	day time.Time,
) (Record, bool) {
	dayKey := day.Format(dateLayout)
	if ctx.InterchangeTemplate == nil || ctx.TrailerByVehicle[vehicleID] == "" ||
		l.hashFraction("form|interchange", driverID, dayKey) >= formInterchangeRate {
		return nil, false
	}
	offset := time.Duration(
		(5 + 12*l.hashFraction("form|interchange-time", driverID, dayKey)) * float64(time.Minute),
	)
	return l.buildFormSubmission(ctx, formSubmissionSpec{
		Template:     ctx.InterchangeTemplate,
		DriverID:     driverID,
		DriverName:   driverName,
		VehicleID:    vehicleID,
		Day:          day,
		Kind:         formSubmissionKindInterchange,
		SubmissionID: formSubmissionID(day, driverID, formSubmissionKindInterchange),
		SubmittedAt:  dayCtx.DrivingEnd.Add(offset).Truncate(time.Second),
	})
}

func routeStopIDAt(route *routeRef, kind string) string {
	stop, ok := route.firstStop()
	if kind == formSubmissionKindPOD {
		stop, ok = route.lastStop()
	}
	if !ok {
		return ""
	}
	return stop.ID
}

func coupledTrailersByVehicle(assets []Record) map[string]string {
	out := make(map[string]string, len(assets))
	for _, asset := range assets {
		if assetType(asset) != assetTypeTrailer {
			continue
		}
		vehicleID := stringValue(asset, fieldSimCoupledVehicleID)
		if vehicleID == "" {
			continue
		}
		if current, exists := out[vehicleID]; !exists || recordID(asset) < current {
			out[vehicleID] = recordID(asset)
		}
	}
	return out
}

func formAPIUserID(templates []Record) string {
	users := formUserIDs(templates)
	ids := make([]string, 0, len(users))
	for id := range users {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func formUserIDs(templates []Record) map[string]struct{} {
	out := map[string]struct{}{}
	for _, template := range templates {
		for _, key := range []string{"createdBy", "updatedBy"} {
			if nestedString(template, key, keyType) == formUserTypeUser {
				if id := nestedString(template, key, keyID); id != "" {
					out[id] = struct{}{}
				}
			}
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
					float64(
						time.Minute,
					),
			),
		).Truncate(time.Second)
	}

	status := formSubmissionStatusCompleted
	if _, requiresApproval := anyAsMap(spec.Template["approvalConfig"]); requiresApproval {
		status = formSubmissionStatusNeedsReview
	}
	record := Record{
		"id":              spec.SubmissionID,
		"title":           stringValue(spec.Template, "title") + " - " + spec.DriverName,
		"status":          status,
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
		"durationMs": spec.SubmittedAt.Sub(createdAt).Milliseconds(),
	}
	location := l.formSubmissionLocation(ctx, &spec)
	if location != nil {
		record["location"] = location
	}
	record["fields"] = l.formFieldInputs(ctx, &spec, location)
	if spec.RouteID != "" {
		record["routeId"] = spec.RouteID
	}
	if spec.RouteStopID != "" {
		record["routeStopId"] = spec.RouteStopID
	}
	return record, true
}

func formSubmissionID(day time.Time, driverID, suffix string) string {
	return deterministicUUID(
		"form-submission",
		day.UTC().Format("20060102"),
		strings.TrimSpace(driverID),
		suffix,
	)
}

type formTemplatePartition struct {
	Generic     []Record
	BOL         Record
	POD         Record
	Interchange Record
}

func partitionFormTemplates(templates []Record) formTemplatePartition {
	out := formTemplatePartition{Generic: make([]Record, 0, len(templates))}
	for _, template := range templates {
		switch stringValue(template, "title") {
		case formTemplateTitleBillOfLading:
			out.BOL = template
		case formTemplateTitleProofOfDelivery:
			out.POD = template
		case formTemplateTitleTrailerInterchange:
			out.Interchange = template
		default:
			out.Generic = append(out.Generic, template)
		}
	}
	return out
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

func (l *LiveSimulator) formFieldInputs(
	ctx *formGenerationContext,
	spec *formSubmissionSpec,
	location map[string]any,
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
			key, value, ok := l.generatedExtendedValue(ctx, spec, field, location, valueHash)
			if !ok {
				continue
			}
			input[key] = value
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
	mediaID := deterministicUUID(submissionID, fieldID, "signature-media")
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

func sortFormSubmissions(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		left := stringutils.FirstNonEmptyTrimmed(
			stringValue(records[i], "submittedAtTime"),
			stringValue(records[i], "updatedAtTime"),
		)
		right := stringutils.FirstNonEmptyTrimmed(
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

func (l *LiveSimulator) generatedExtendedValue(
	ctx *formGenerationContext,
	spec *formSubmissionSpec,
	field Record,
	location map[string]any,
	valueHash float64,
) (key string, value map[string]any, ok bool) {
	fieldID := stringValue(field, keyID)
	fieldType := stringValue(field, keyType)
	switch fieldType {
	case formFieldTypeDateTime:
		at := spec.SubmittedAt.Add(-time.Duration(2+6*valueHash) * time.Minute)
		rendered := map[string]any{keyType: formFieldTypeDateTime, keyValue: at.UTC().Format(time.RFC3339)}
		return formValueKeys[fieldType], rendered, true
	case formFieldTypeAsset:
		trailerID := ctx.TrailerByVehicle[spec.VehicleID]
		asset, exists := l.fleet().assetByID[trailerID]
		if !exists {
			return "", nil, false
		}
		return formValueKeys[fieldType], map[string]any{keyAsset: formTrackedAsset(asset)}, true
	case formFieldTypeGeofence:
		circle, found := nearestGeofence(ctx.Geofences, location)
		if !found {
			return "", nil, false
		}
		return formValueKeys[fieldType], map[string]any{"geofence": map[string]any{
			"entryType":      formEntryTypeTracked,
			keyID:            circle.AddressID,
			"address":        circle.FormattedAddress,
			fieldExternalIDs: cloneMap(circle.ExternalIDs),
		}}, true
	case formFieldTypeBarcode:
		seal := fmt.Sprintf("SL-%06d", int(valueHash*1000000)%1000000)
		return formValueKeys[fieldType], map[string]any{
			"barcodes": []any{map[string]any{keyValue: seal}},
		}, true
	case formFieldTypeMedia:
		media := make([]any, 0, formInterchangePhotoCount)
		for idx := range formInterchangePhotoCount {
			media = append(media, formMediaRecord(
				deterministicUUID(spec.SubmissionID, fieldID, "photo", strconv.Itoa(idx)),
			))
		}
		return formValueKeys[fieldType], map[string]any{"mediaList": media}, true
	case formFieldTypePerson:
		if ctx.APIUserID == "" {
			return "", nil, false
		}
		return formValueKeys[fieldType], map[string]any{"person": map[string]any{
			"entryType":         formEntryTypeTracked,
			"polymorphicUserId": map[string]any{keyID: ctx.APIUserID, keyType: formUserTypeUser},
		}}, true
	case formFieldTypeTable:
		return formValueKeys[fieldType], l.generatedTireTable(spec, field), true
	default:
		return "", nil, false
	}
}

func (l *LiveSimulator) generatedTireTable(spec *formSubmissionSpec, field Record) map[string]any {
	columns := make([]any, 0, 3)
	defs := make([]Record, 0, 3)
	for _, raw := range listOf(field["columns"]) {
		column, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		defs = append(defs, Record(column))
		columns = append(columns, map[string]any{
			keyID:   column[keyID],
			"label": column["label"],
			keyType: column[keyType],
		})
	}
	rows := make([]any, 0, len(formTirePositions))
	for idx, position := range formTirePositions {
		depth := math.Round(6 + 8*l.hashFraction("form|tread", spec.SubmissionID, position))
		cells := make([]any, 0, len(defs))
		for _, column := range defs {
			cell := map[string]any{keyID: stringValue(column, keyID), keyType: stringValue(column, keyType)}
			switch stringValue(column, keyType) {
			case formFieldTypeText:
				cell["textValue"] = map[string]any{keyValue: position}
			case formFieldTypeNumber:
				cell["numberValue"] = map[string]any{keyValue: depth}
			case formFieldTypeMultipleChoice:
				choice := 0
				if depth < 8 {
					choice = 1
				}
				option, ok := formOptionAt(column, choice)
				if !ok {
					continue
				}
				cell["multipleChoiceValue"] = map[string]any{
					keyValue:  stringValue(option, "label"),
					"valueId": stringValue(option, keyID),
				}
			default:
				continue
			}
			cells = append(cells, cell)
		}
		rows = append(rows, map[string]any{
			keyID:   deterministicUUID(spec.SubmissionID, "tire-row", strconv.Itoa(idx)),
			"cells": cells,
		})
	}
	return map[string]any{"columns": columns, "rows": rows}
}

func nearestGeofence(circles []geofenceCircle, location map[string]any) (geofenceCircle, bool) {
	if len(circles) == 0 {
		return geofenceCircle{}, false
	}
	latitude, latOK := location[keyLatitude].(float64)
	longitude, lonOK := location[keyLongitude].(float64)
	best := 0
	if latOK && lonOK {
		bestDistance := math.Inf(1)
		for idx := range circles {
			distance := haversineMeters(latitude, longitude, circles[idx].Latitude, circles[idx].Longitude)
			if distance < bestDistance {
				best, bestDistance = idx, distance
			}
		}
	}
	return circles[best], true
}
