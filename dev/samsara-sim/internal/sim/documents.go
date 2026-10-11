package sim

import (
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	documentStateSubmitted = "submitted"
	documentStateRequired  = "required"
	documentQueryCreated   = "created"
	documentQueryUpdated   = "updated"
	documentMaxNotes       = 2000
	documentLookback       = 120 * 24 * time.Hour
	documentMediaURLPrefix = "https://samsara-driver-media-upload.s3.us-west-2.amazonaws.com/"
	documentPDFURLPrefix   = "https://samsara-driver-document-pdfs.s3.us-west-2.amazonaws.com/"
	documentPDFProcessing  = time.Second
	documentPDFCompleted   = 4 * time.Second
	documentPDFRequested   = "requested"
	documentPDFIsRunning   = "processing"
	documentPDFIsComplete  = "completed"
	documentEventSubmitted = "DocumentSubmitted"

	documentFieldPhoto          = "photo"
	documentFieldString         = "string"
	documentFieldNumber         = "number"
	documentFieldMultipleChoice = "multipleChoice"
	documentFieldSignature      = "signature"
	documentFieldDateTime       = "dateTime"
	documentFieldScanned        = "scannedDocument"
	documentFieldBarcode        = "barcode"

	documentTypeBOL     = "Bill of Lading"
	documentTypePOD     = "Proof of Delivery"
	documentTypeFuel    = "Fuel Receipt"
	documentTypeLumper  = "Lumper Receipt"
	documentTypeScale   = "Scale Ticket"
	documentScaleRate   = 0.3
	documentFuelRate    = 0.55
	documentLumperRate  = 0.25
	fieldSimDeleted     = "simDeleted"
	fieldDocumentType   = "documentType"
	fieldDocumentTypeID = "documentTypeId"
	fieldRouteStop      = "routeStop"
	fieldRoute          = "route"
	fieldConditional    = "conditionalFieldSections"
)

var (
	documentFieldTypes = []string{
		documentFieldPhoto, documentFieldString, documentFieldNumber,
		documentFieldMultipleChoice, documentFieldSignature, documentFieldDateTime,
		documentFieldScanned, documentFieldBarcode,
	}
	documentValueKeys = map[string]string{
		documentFieldPhoto:          "photoValue",
		documentFieldString:         "stringValue",
		documentFieldNumber:         "numberValue",
		documentFieldMultipleChoice: "multipleChoiceValue",
		documentFieldSignature:      "signatureValue",
		documentFieldDateTime:       "dateTimeValue",
		documentFieldScanned:        "scannedDocumentValue",
		documentFieldBarcode:        "barcodeValue",
	}
	documentLumperServices = []string{
		"Capstone Logistics",
		"NFI Unloading Services",
		"Pinnacle Unloading",
		"RLS Logistics Lumpers",
	}
	documentScaleLocations = []string{
		"CAT Scale #1173, I-35 Exit 250, Georgetown, TX",
		"CAT Scale #841, I-45 Exit 64, Conroe, TX",
		"CAT Scale #1520, I-10 Exit 583, Seguin, TX",
		"CAT Scale #977, I-20 Exit 465, Weatherford, TX",
	}
	documentDamageNotes = []string{
		"Two cartons crushed on pallet 3; receiver noted on POD",
		"Shrink wrap torn on one pallet, product intact",
		"Corner of top tier damaged by forklift at dock",
	}
)

func (s *Server) registerDocumentRoutes() {
	s.mux.HandleFunc("GET /fleet/document-types", s.handleDocumentTypeList)
	s.mux.HandleFunc("GET /fleet/documents", s.handleDocumentList)
	s.mux.HandleFunc("POST /fleet/documents", s.handleDocumentCreate)
	s.mux.HandleFunc("GET /fleet/documents/{id}", s.handleDocumentGet)
	s.mux.HandleFunc("DELETE /fleet/documents/{id}", s.handleDocumentDelete)
	s.mux.HandleFunc("POST /fleet/documents/pdfs", s.handleDocumentPDFCreate)
	s.mux.HandleFunc("GET /fleet/documents/pdfs/{id}", s.handleDocumentPDFGet)
}

func (s *Server) handleDocumentTypeList(writer http.ResponseWriter, request *http.Request) {
	types, err := s.store.List(ResourceDocumentTypes)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondPage(writer, request, types, "|document-type-list")
}

func documentTypeByID(types []Record, id string) (Record, bool) {
	for _, docType := range types {
		if recordID(docType) == id {
			return docType, true
		}
	}
	return nil, false
}

func documentTypeByName(types []Record, name string) (Record, bool) {
	for _, docType := range types {
		if stringValue(docType, keyName) == name {
			return docType, true
		}
	}
	return nil, false
}

func documentTypeTiny(docType Record) map[string]any {
	return map[string]any{keyID: recordID(docType), keyName: stringValue(docType, keyName)}
}

func documentConditionalSections(docType Record) []any {
	sections, ok := docType[fieldConditional].([]any)
	if !ok {
		return []any{}
	}
	return cloneAny(sections).([]any)
}

type documentState struct {
	stored  []Record
	deleted map[string]struct{}
	live    map[string]Record
}

func (s *Server) documentState() (documentState, error) {
	records, err := s.store.List(ResourceDocuments)
	if err != nil {
		return documentState{}, err
	}
	state := documentState{
		stored:  make([]Record, 0, len(records)),
		deleted: make(map[string]struct{}),
		live:    make(map[string]Record, len(records)),
	}
	for _, record := range records {
		if record[fieldSimDeleted] == true {
			state.deleted[recordID(record)] = struct{}{}
			continue
		}
		state.stored = append(state.stored, record)
		state.live[recordID(record)] = record
	}
	return state, nil
}

func (s *Server) documentsInWindow(now, start, end time.Time) ([]Record, error) {
	state, err := s.documentState()
	if err != nil {
		return nil, err
	}
	out := append(make([]Record, 0, len(state.stored)), state.stored...)
	if s.live == nil {
		return out, nil
	}
	for _, record := range s.live.GeneratedDocuments(now, start, end) {
		id := recordID(record)
		if _, gone := state.deleted[id]; gone {
			continue
		}
		if _, overridden := state.live[id]; overridden {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func (s *Server) findDocument(now time.Time, id string) (Record, bool, error) {
	state, err := s.documentState()
	if err != nil {
		return nil, false, err
	}
	if record, ok := state.live[id]; ok {
		return record, true, nil
	}
	if _, gone := state.deleted[id]; gone || s.live == nil {
		return nil, false, nil
	}
	for _, record := range s.live.GeneratedDocuments(now, now.Add(-documentLookback), now) {
		if recordID(record) == id {
			return record, false, nil
		}
	}
	return nil, false, nil
}

func (s *Server) handleDocumentList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if startTime == nil || endTime == nil {
		s.writeError(writer, ErrTimeRangeRequired)
		return
	}
	queryBy := strings.TrimSpace(values.Get("queryBy"))
	if queryBy == "" {
		queryBy = documentQueryCreated
	}
	if queryBy != documentQueryCreated && queryBy != documentQueryUpdated {
		s.writeError(writer, invalidParameter("queryBy", "must be `created` or `updated`"))
		return
	}
	timeField := fieldCreatedAtTime
	if queryBy == documentQueryUpdated {
		timeField = fieldUpdatedAtTime
	}
	typeID := strings.TrimSpace(values.Get(fieldDocumentTypeID))
	now := s.simNow()
	pool, err := s.documentsInWindow(now, *startTime, *endTime)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	selected := make([]Record, 0, len(pool))
	for _, record := range pool {
		at, parseErr := parseRFC3339(stringValue(record, timeField))
		if parseErr != nil || at.Before(*startTime) || at.After(*endTime) {
			continue
		}
		if typeID != "" && nestedString(record, fieldDocumentType, keyID) != typeID {
			continue
		}
		selected = append(selected, record)
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left, right := stringValue(selected[i], timeField), stringValue(selected[j], timeField)
		if left != right {
			return left < right
		}
		return recordID(selected[i]) < recordID(selected[j])
	})
	s.respondPage(writer, request, selected, "|document-list")
}

func (s *Server) handleDocumentGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	record, _, err := s.findDocument(s.simNow(), id)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if record == nil {
		s.writeError(writer, notFound("document", id))
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|document-get", map[string]any{keyData: record})
}

func (s *Server) handleDocumentDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	record, stored, err := s.findDocument(now, id)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if record == nil {
		s.writeError(writer, notFound("document", id))
		return
	}
	err = s.store.Transact(func(tx *storeTx) error {
		if stored {
			return tx.remove(ResourceDocuments, id)
		}
		return tx.replace(
			ResourceDocuments,
			append(tx.records(ResourceDocuments), Record{keyID: id, fieldSimDeleted: true}),
		)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

type documentWriteContext struct {
	view  *fleetView
	types []Record
	stops []routeRef
}

func (s *Server) handleDocumentCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	types, err := s.store.List(ResourceDocumentTypes)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	ctx := &documentWriteContext{view: s.fleetView(), types: types}
	if s.live != nil {
		ctx.stops = s.live.routeRefs(now)
	}
	record, err := ctx.build(body)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	var created Record
	err = s.store.Transact(func(tx *storeTx) error {
		inserted, insertErr := tx.insert(ResourceDocuments, record, now)
		created = cloneRecord(inserted)
		return insertErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|document-create", map[string]any{keyData: created})
}

func (c *documentWriteContext) build(body Record) (Record, error) {
	typeID, err := requiredString(body, fieldDocumentTypeID, "body")
	if err != nil {
		return nil, invalidField(fieldDocumentTypeID, "is required")
	}
	docType, ok := documentTypeByID(c.types, typeID)
	if !ok {
		return nil, missingReference("document type", typeID)
	}
	driverRef, err := requiredString(body, "driverId", "body")
	if err != nil {
		return nil, invalidField("driverId", "is required")
	}
	driver, idx := findRecordByRef(c.view.snap.drivers, driverRef, nil)
	if idx < 0 {
		return nil, missingReference(keyDriver, driverRef)
	}
	state := documentStateRequired
	if raw, present := body[keyState]; present && raw != nil {
		value, isString := raw.(string)
		if !isString || (value != documentStateSubmitted && value != documentStateRequired) {
			return nil, invalidField(keyState, "must be one of `submitted`, `required`")
		}
		state = value
	}
	record := Record{
		fieldDocumentType: documentTypeTiny(docType),
		keyDriver:         c.view.driverRef(recordID(driver)),
		keyState:          state,
		fieldConditional:  documentConditionalSections(docType),
		keyName:           stringValue(docType, keyName),
	}
	if err = c.applyOptional(record, body); err != nil {
		return nil, err
	}
	fields, err := sanitizeDocumentFields(docType, body["fields"], state == documentStateSubmitted)
	if err != nil {
		return nil, err
	}
	record["fields"] = fields
	return record, nil
}

func (c *documentWriteContext) applyOptional(record, body Record) error {
	if raw, present := body[keyName]; present && raw != nil {
		name, ok := raw.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return invalidField(keyName, "must be a non-empty string")
		}
		record[keyName] = strings.TrimSpace(name)
	}
	if raw, present := body[keyNotes]; present && raw != nil {
		notes, err := sanitizeString(raw, &fieldRule{MaxLen: documentMaxNotes}, keyNotes)
		if err != nil {
			return err
		}
		record[keyNotes] = notes
	}
	if raw, present := body["vehicleId"]; present && raw != nil {
		ref, ok := raw.(string)
		if !ok {
			return invalidField("vehicleId", "must be a string")
		}
		vehicle, idx := findRecordByRef(c.view.vehicleRecords(), ref, vehicleAutoExternalIDs)
		if idx < 0 {
			return missingReference(keyVehicle, ref)
		}
		record[keyVehicle] = c.view.vehicleRef(recordID(vehicle))
	}
	if raw, present := body[fieldRouteStopID]; present && raw != nil {
		ref, ok := raw.(string)
		if !ok {
			return invalidField(fieldRouteStopID, "must be a string")
		}
		match, found := findRouteStop(c.stops, ref, true)
		if !found {
			return missingReference("route stop", ref)
		}
		record[fieldRoute] = match.Route.tiny()
		record[fieldRouteStop] = match.Stop.tiny()
	}
	return nil
}

func sanitizeDocumentFields(docType Record, raw any, enforceRequired bool) ([]any, error) {
	definitions := listOf(docType["fieldTypes"])
	byLabel := make(map[string]Record, len(definitions))
	for _, item := range definitions {
		if def, ok := anyAsMap(item); ok {
			byLabel[stringOf(def["label"])] = Record(def)
		}
	}
	out := make([]any, 0, len(definitions))
	seen := make(map[string]struct{}, len(definitions))
	if raw != nil {
		items, ok := raw.([]any)
		if !ok {
			return nil, invalidField("fields", "must be an array")
		}
		for idx, item := range items {
			field, err := sanitizeDocumentField(byLabel, item, fmt.Sprintf("fields[%d]", idx))
			if err != nil {
				return nil, err
			}
			label := stringOf(field["label"])
			if _, dup := seen[label]; dup {
				return nil, invalidField(fmt.Sprintf("fields[%d].label", idx), "appears more than once")
			}
			seen[label] = struct{}{}
			out = append(out, field)
		}
	}
	if !enforceRequired {
		return out, nil
	}
	for _, item := range definitions {
		def, ok := anyAsMap(item)
		if !ok || def["requiredField"] != true {
			continue
		}
		if _, provided := seen[stringOf(def["label"])]; !provided {
			return nil, invalidField(
				"fields",
				fmt.Sprintf("a submitted document needs the required field %q", stringOf(def["label"])),
			)
		}
	}
	return out, nil
}

func sanitizeDocumentField(byLabel map[string]Record, raw any, path string) (map[string]any, error) {
	field, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(path, "must be an object")
	}
	label, err := requiredString(field, "label", path)
	if err != nil {
		return nil, err
	}
	fieldType, err := requiredEnum(field, keyType, path, documentFieldTypes)
	if err != nil {
		return nil, err
	}
	def, exists := byLabel[label]
	if !exists {
		return nil, invalidField(path+".label", fmt.Sprintf("%q is not a field of this document type", label))
	}
	if expected := stringOf(def["fieldType"]); expected != fieldType {
		return nil, invalidField(path+".type", fmt.Sprintf("must be %q for field %q", expected, label))
	}
	rawValue, present := field[keyValue]
	if !present || rawValue == nil {
		if def["requiredField"] == true {
			return nil, invalidField(path+".value", "is required for "+label)
		}
		return map[string]any{"label": label, keyType: fieldType, keyValue: map[string]any{}}, nil
	}
	value, ok := anyAsMap(rawValue)
	if !ok {
		return nil, invalidField(path+".value", "must be an object")
	}
	rendered, err := sanitizeDocumentValue(def, fieldType, value, path+".value")
	if err != nil {
		return nil, err
	}
	return map[string]any{"label": label, keyType: fieldType, keyValue: rendered}, nil
}

func sanitizeDocumentValue(
	def Record,
	fieldType string,
	value map[string]any,
	path string,
) (map[string]any, error) {
	expected := documentValueKeys[fieldType]
	for otherType, key := range documentValueKeys {
		if key == expected {
			continue
		}
		if _, present := value[key]; present {
			return nil, invalidField(path+"."+key, "is only present for "+otherType+" fields")
		}
	}
	raw, present := value[expected]
	if !present || raw == nil {
		return nil, invalidField(path+"."+expected, "is required for "+fieldType+" fields")
	}
	valuePath := path + "." + expected
	var (
		sanitized any
		err       error
	)
	switch fieldType {
	case documentFieldString:
		sanitized, err = sanitizeString(raw, &fieldRule{}, valuePath)
	case documentFieldNumber:
		sanitized, err = sanitizeDocumentNumber(def, raw, valuePath)
	case documentFieldMultipleChoice:
		sanitized, err = sanitizeDocumentChoices(def, raw, valuePath)
	case documentFieldSignature:
		sanitized, err = sanitizeDocumentSignature(raw, valuePath)
	case documentFieldDateTime:
		sanitized, err = sanitizeDocumentDateTime(raw, valuePath)
	case documentFieldPhoto, documentFieldScanned:
		sanitized, err = sanitizeDocumentMedia(raw, valuePath)
	case documentFieldBarcode:
		sanitized, err = sanitizeDocumentBarcodes(raw, valuePath)
	default:
		err = invalidField(valuePath, "is not supported")
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{expected: sanitized}, nil
}

func sanitizeDocumentNumber(def Record, raw any, path string) (float64, error) {
	number, err := sanitizeNumber(raw, path)
	if err != nil {
		return 0, err
	}
	places, has := int64Value(nestedAny(def, "numberFieldTypeMetaData", "numberOfDecimalPlaces"))
	if has && decimalPlaces(number) > int(places) {
		return 0, invalidField(path, fmt.Sprintf("allows at most %d decimal places", places))
	}
	return number, nil
}

func documentChoiceLabels(def Record) []string {
	options := listOf(def["multipleChoiceFieldTypeMetaData"])
	out := make([]string, 0, len(options))
	for _, raw := range options {
		if option, ok := anyAsMap(raw); ok {
			out = append(out, stringOf(option["label"]))
		}
	}
	return out
}

func sanitizeDocumentChoices(def Record, raw any, path string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array")
	}
	labels := documentChoiceLabels(def)
	selected := make(map[string]bool, len(items))
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		choice, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		label, err := requiredString(choice, keyValue, itemPath)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(labels, label) {
			return nil, invalidField(itemPath+".value", fmt.Sprintf("%q is not an option of this field", label))
		}
		chosen, isBool := choice["selected"].(bool)
		if !isBool {
			return nil, invalidField(itemPath+".selected", "must be a boolean")
		}
		selected[label] = selected[label] || chosen
	}
	count := 0
	out := make([]any, 0, len(labels))
	for _, label := range labels {
		if selected[label] {
			count++
		}
		out = append(out, map[string]any{"selected": selected[label], keyValue: label})
	}
	if count > 1 {
		return nil, invalidField(path, "only one choice may be selected")
	}
	return out, nil
}

func sanitizeDocumentSignature(raw any, path string) (map[string]any, error) {
	signature, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(path, "must be an object")
	}
	out := map[string]any{}
	for _, key := range []string{keyID, keyName, "url"} {
		if value, present := signature[key]; present && value != nil {
			text, isString := value.(string)
			if !isString {
				return nil, invalidField(path+"."+key, "must be a string")
			}
			out[key] = text
		}
	}
	if value, present := signature["signedAtTime"]; present && value != nil {
		signedAt, err := sanitizeTime(value, path+".signedAtTime")
		if err != nil {
			return nil, err
		}
		out["signedAtTime"] = signedAt
	}
	return out, nil
}

func sanitizeDocumentDateTime(raw any, path string) (map[string]any, error) {
	value, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(path, "must be an object")
	}
	at, err := sanitizeTime(value["dateTime"], path+".dateTime")
	if err != nil {
		return nil, err
	}
	return map[string]any{"dateTime": at}, nil
}

func sanitizeDocumentMedia(raw any, path string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array")
	}
	out := make([]any, 0, len(items))
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		media, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		entry := map[string]any{}
		for _, key := range []string{keyID, "url"} {
			text, err := requiredString(media, key, itemPath)
			if err != nil {
				return nil, err
			}
			entry[key] = text
		}
		out = append(out, entry)
	}
	return out, nil
}

func sanitizeDocumentBarcodes(raw any, path string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array")
	}
	out := make([]any, 0, len(items))
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		barcode, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		entry := map[string]any{}
		for _, key := range []string{"barcodeType", "barcodeValue"} {
			text, err := requiredString(barcode, key, itemPath)
			if err != nil {
				return nil, err
			}
			entry[key] = text
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Server) handleDocumentPDFCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	documentID, err := requiredString(body, "documentId", "body")
	if err != nil {
		s.writeError(writer, invalidField("documentId", "is required"))
		return
	}
	now := s.simNow()
	record, _, err := s.findDocument(now, documentID)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if record == nil {
		s.writeError(writer, notFound("document", documentID))
		return
	}
	var job Record
	err = s.store.Transact(func(tx *storeTx) error {
		inserted, insertErr := tx.insert(ResourceDocumentPDFs, Record{
			"documentId":      documentID,
			"requestedAtTime": now.UTC().Format(time.RFC3339),
		}, now)
		job = cloneRecord(inserted)
		return insertErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|document-pdf-create", map[string]any{
		keyData: map[string]any{keyID: recordID(job), "documentId": documentID},
	})
}

func (s *Server) handleDocumentPDFGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	job, err := s.store.Get(ResourceDocumentPDFs, id)
	if err != nil {
		s.writeError(writer, notFound("document PDF", id))
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|document-pdf-get", map[string]any{
		keyData: renderDocumentPDF(job, s.simNow()),
	})
}

func renderDocumentPDF(job Record, now time.Time) map[string]any {
	requestedAt, _ := parseRFC3339(stringValue(job, "requestedAtTime"))
	out := map[string]any{
		keyID:             recordID(job),
		"documentId":      stringValue(job, "documentId"),
		"requestedAtTime": requestedAt.Format(time.RFC3339),
		"jobStatus":       documentPDFRequested,
	}
	elapsed := now.Sub(requestedAt)
	switch {
	case elapsed >= documentPDFCompleted:
		out["jobStatus"] = documentPDFIsComplete
		out["completedAtTime"] = requestedAt.Add(documentPDFCompleted).Format(time.RFC3339)
		out["downloadDocumentPdfUrl"] = documentPDFURLPrefix +
			strconv.FormatInt(webhookOrgID, 10) + "/" + recordID(job) + ".pdf"
	case elapsed >= documentPDFProcessing:
		out["jobStatus"] = documentPDFIsRunning
	}
	return out
}

func (s *Server) dispatchDocumentEvents(request *http.Request, at time.Time) {
	if s.live == nil {
		return
	}
	windowStart := s.documentWindow.nextStart(at, defaultAssetLookback, geofenceMaxWindowLookback)
	state, err := s.documentState()
	if err != nil {
		return
	}
	for _, record := range s.live.GeneratedDocuments(at, windowStart, at) {
		id := recordID(record)
		if _, gone := state.deleted[id]; gone {
			continue
		}
		s.dispatchEventOnce(
			request,
			documentEventSubmitted+"|"+id,
			documentEventSubmitted,
			map[string]any{"document": cloneRecord(record)},
		)
	}
}

type documentGenerationContext struct {
	now      time.Time
	types    []Record
	roster   map[string]driverRoster
	routes   map[string]routeRef
	snapshot *fleetSnapshot
}

type documentDaySpec struct {
	docType  Record
	driverID string
	vehicle  string
	route    *routeRef
	stop     *routeStopRef
	kind     string
	at       time.Time
	dayKey   string
}

func (l *LiveSimulator) GeneratedDocuments(now, start, end time.Time) []Record {
	if end.After(now) {
		end = now
	}
	if start.Before(telemetryEpoch) {
		start = telemetryEpoch
	}
	if end.Before(start) {
		return []Record{}
	}
	types, err := l.store.List(ResourceDocumentTypes)
	if err != nil || len(types) == 0 {
		return []Record{}
	}
	ctx := &documentGenerationContext{
		now:      now,
		types:    types,
		roster:   l.loadDriverRoster(),
		routes:   routeRefsByDriver(l.routeRefs(now)),
		snapshot: l.fleet(),
	}
	ids := make([]string, 0, len(ctx.roster))
	for driverID := range ctx.roster {
		if driver, ok := ctx.snapshot.driverByID[driverID]; ok && !driverDeactivated(driver) {
			ids = append(ids, driverID)
		}
	}
	sort.Strings(ids)
	startDay := start.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	endDay := end.Truncate(24 * time.Hour)
	out := make([]Record, 0, len(ids)*4)
	for _, driverID := range ids {
		for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
			for _, record := range l.driverDayDocuments(ctx, driverID, day) {
				created, parseErr := parseRFC3339(stringValue(record, fieldCreatedAtTime))
				if parseErr != nil || created.Before(start) || created.After(end) {
					continue
				}
				out = append(out, record)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, right := stringValue(out[i], fieldCreatedAtTime), stringValue(out[j], fieldCreatedAtTime)
		if left != right {
			return left < right
		}
		return recordID(out[i]) < recordID(out[j])
	})
	return out
}

func (l *LiveSimulator) driverDayDocuments(
	ctx *documentGenerationContext,
	driverID string,
	day time.Time,
) []Record {
	entry := ctx.roster[driverID]
	vehicleID := strings.TrimSpace(entry.VehicleID)
	dayCtx := l.buildDailyEventContext(driverID, vehicleID, day)
	span := dayCtx.DrivingEnd.Sub(dayCtx.DrivingStart)
	if span <= 0 {
		return nil
	}
	dayKey := day.Format(dateLayout)
	route := ctx.routes[driverID]
	first, hasFirst := route.firstStop()
	last, hasLast := route.lastStop()
	hash := func(parts ...string) float64 {
		return l.hashFraction(append([]string{"doc", driverID, dayKey}, parts...)...)
	}
	minutes := func(base, spread float64, key string) time.Duration {
		return time.Duration((base + spread*hash(key)) * float64(time.Minute))
	}
	specs := make([]documentDaySpec, 0, 5)
	add := func(typeName, kind string, at time.Time, stop *routeStopRef) {
		docType, ok := documentTypeByName(ctx.types, typeName)
		if !ok || at.After(ctx.now) {
			return
		}
		specs = append(specs, documentDaySpec{
			docType:  docType,
			driverID: driverID,
			vehicle:  vehicleID,
			route:    &route,
			stop:     stop,
			kind:     kind,
			at:       at.Truncate(time.Second),
			dayKey:   dayKey,
		})
	}
	pickupAt := dayCtx.DrivingStart.Add(minutes(8, 22, "bol-at"))
	add(documentTypeBOL, "bol", pickupAt, stopPtr(first, hasFirst))
	if hash("scale") < documentScaleRate {
		add(documentTypeScale, "scale", pickupAt.Add(minutes(25, 30, "scale-at")), nil)
	}
	if hash("fuel") < documentFuelRate {
		fraction := 0.45 + 0.15*hash("fuel-at")
		add(documentTypeFuel, "fuel", dayCtx.DrivingStart.Add(time.Duration(fraction*float64(span))), nil)
	}
	deliveryAt := dayCtx.DrivingEnd.Add(-minutes(2, 13, "pod-at"))
	if hash("lumper") < documentLumperRate {
		add(documentTypeLumper, "lumper", deliveryAt.Add(-minutes(25, 15, "lumper-at")), stopPtr(last, hasLast))
	}
	add(documentTypePOD, "pod", deliveryAt, stopPtr(last, hasLast))

	out := make([]Record, 0, len(specs))
	for idx := range specs {
		out = append(out, l.generatedDocument(ctx, &specs[idx]))
	}
	return out
}

func stopPtr(stop routeStopRef, ok bool) *routeStopRef {
	if !ok {
		return nil
	}
	return &stop
}

func (l *LiveSimulator) generatedDocument(ctx *documentGenerationContext, spec *documentDaySpec) Record {
	id := deterministicUUID("document", spec.dayKey, spec.driverID, spec.kind)
	stamp := spec.at.UTC().Format(time.RFC3339)
	view := &fleetView{live: l, snap: ctx.snapshot, now: ctx.now}
	record := Record{
		keyID:              id,
		fieldDocumentType:  documentTypeTiny(spec.docType),
		keyDriver:          view.driverRef(spec.driverID),
		keyState:           documentStateSubmitted,
		fieldConditional:   documentConditionalSections(spec.docType),
		fieldCreatedAtTime: stamp,
		fieldUpdatedAtTime: stamp,
	}
	if spec.vehicle != "" {
		record[keyVehicle] = view.vehicleRef(spec.vehicle)
	}
	if spec.route != nil && spec.route.ID != "" {
		record[fieldRoute] = spec.route.tiny()
	}
	if spec.stop != nil {
		record[fieldRouteStop] = spec.stop.tiny()
	}
	values := l.documentValues(spec, id)
	fields := make([]any, 0, len(listOf(spec.docType["fieldTypes"])))
	for _, raw := range listOf(spec.docType["fieldTypes"]) {
		def, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		label := stringOf(def["label"])
		value, has := values.fields[label]
		if !has {
			continue
		}
		fields = append(fields, map[string]any{"label": label, keyType: def["fieldType"], keyValue: value})
	}
	record["fields"] = fields
	record[keyName] = values.name
	if values.notes != "" {
		record[keyNotes] = values.notes
	}
	return record
}

type documentValueSet struct {
	name   string
	notes  string
	fields map[string]map[string]any
}

func (l *LiveSimulator) documentValues(spec *documentDaySpec, documentID string) documentValueSet {
	hash := func(parts ...string) float64 {
		return l.hashFraction(append([]string{"doc-value", spec.driverID, spec.dayKey, spec.kind}, parts...)...)
	}
	stamp := spec.at.UTC().Format(time.RFC3339)
	media := func(label string, count int) map[string]any {
		key := documentValueKeys[documentFieldPhoto]
		if label == "Signed BOL" || label == "Signed POD" {
			key = documentValueKeys[documentFieldScanned]
		}
		items := make([]any, 0, count)
		for idx := range count {
			mediaID := deterministicUUID(documentID, label, strconv.Itoa(idx))
			items = append(items, map[string]any{
				keyID: mediaID,
				"url": documentMediaURLPrefix + strconv.FormatInt(webhookOrgID, 10) + "/" + mediaID + ".jpg",
			})
		}
		return map[string]any{key: items}
	}
	number := func(value float64) map[string]any { return map[string]any{"numberValue": value} }
	text := func(value string) map[string]any { return map[string]any{"stringValue": value} }
	choices := func(labels []string, selected string) map[string]any {
		items := make([]any, 0, len(labels))
		for _, label := range labels {
			items = append(items, map[string]any{"selected": label == selected, keyValue: label})
		}
		return map[string]any{"multipleChoiceValue": items}
	}
	signature := func(label, name string) map[string]any {
		signatureID := deterministicUUID(documentID, label, "signature")
		return map[string]any{"signatureValue": map[string]any{
			keyID:          signatureID,
			keyName:        name,
			"signedAtTime": stamp,
			"url": documentMediaURLPrefix + strconv.FormatInt(webhookOrgID, 10) + "/" +
				signatureID + ".png",
		}}
	}
	stopName := ""
	if spec.stop != nil {
		stopName = spec.stop.Name
	}
	out := documentValueSet{fields: map[string]map[string]any{}}
	switch stringValue(spec.docType, keyName) {
	case documentTypeBOL:
		bolNumber := fmt.Sprintf("BOL-%07d", int(hash("number")*10_000_000))
		out.name = bolNumber
		out.fields["BOL Number"] = text(bolNumber)
		out.fields["Signed BOL"] = media("Signed BOL", 1)
		out.fields["Pieces"] = number(math.Round(4 + 22*hash("pieces")))
		out.fields["Weight (lbs)"] = number(math.Round(8000 + 36000*hash("weight")))
		out.fields["Seal Number"] = map[string]any{"barcodeValue": []any{map[string]any{
			"barcodeType":  "org.iso.Code128",
			"barcodeValue": fmt.Sprintf("SL-%06d", int(hash("seal")*1_000_000)),
		}}}
		out.fields["Shipper Signature"] = signature("Shipper Signature", formCatalogValue(formReceiverNames, hash("shipper")))
	case documentTypePOD:
		receiver := formCatalogValue(formReceiverNames, hash("receiver"))
		exception := documentException(hash("exception"))
		out.name = "POD - " + stringutils.FirstNonEmptyTrimmed(stopName, spec.dayKey)
		out.fields["Signed POD"] = media("Signed POD", 1)
		out.fields["Receiver Name"] = text(receiver)
		out.fields["Delivered At"] = map[string]any{"dateTimeValue": map[string]any{"dateTime": stamp}}
		out.fields["Receiver Signature"] = signature("Receiver Signature", receiver)
		out.fields["Delivery Exception"] = choices(
			documentChoiceLabels(fieldDefinition(spec.docType, "Delivery Exception")),
			exception,
		)
		if exception == "Damage" {
			out.fields["Damage Photos"] = media("Damage Photos", 2)
			note := formCatalogValue(documentDamageNotes, hash("damage-note"))
			out.fields["Damage Notes"] = text(note)
			out.notes = note
		} else if exception != "None" {
			out.notes = exception + " noted by receiver at delivery"
		}
	case documentTypeFuel:
		gallons := round(60+120*hash("gallons"), 3)
		price := round(3.15+1.1*hash("price"), 3)
		out.name = "Fuel Receipt - " + spec.dayKey
		out.fields["Receipt Photo"] = media("Receipt Photo", 1)
		out.fields["Fuel Type"] = choices(
			documentChoiceLabels(fieldDefinition(spec.docType, "Fuel Type")),
			"Diesel",
		)
		out.fields["Gallons"] = number(gallons)
		out.fields["Price per Gallon"] = number(price)
		out.fields["Total Amount"] = number(round(gallons*price, 2))
	case documentTypeLumper:
		methods := documentChoiceLabels(fieldDefinition(spec.docType, "Payment Method"))
		method := ""
		if len(methods) > 0 {
			method = methods[min(int(hash("method")*float64(len(methods))), len(methods)-1)]
		}
		out.name = "Lumper Receipt - " + stringutils.FirstNonEmptyTrimmed(stopName, spec.dayKey)
		out.fields["Receipt Photo"] = media("Receipt Photo", 1)
		out.fields["Lumper Service"] = text(formCatalogValue(documentLumperServices, hash("service")))
		out.fields["Amount"] = number(round(85+240*hash("amount"), 2))
		out.fields["Payment Method"] = choices(methods, method)
		if method == "Comchek" || method == "EFS Check" {
			out.fields["Check Number"] = text(fmt.Sprintf("%010d", int(hash("check")*10_000_000_000)))
		}
	case documentTypeScale:
		steer := math.Round(10_400 + 1_400*hash("steer"))
		drive := math.Round(26_000 + 8_000*hash("drive"))
		trailer := math.Round(24_000 + 9_500*hash("trailer"))
		out.name = "Scale Ticket - " + spec.dayKey
		out.fields["Ticket Photo"] = media("Ticket Photo", 1)
		out.fields["Scale Location"] = text(formCatalogValue(documentScaleLocations, hash("scale")))
		out.fields["Steer Axle (lbs)"] = number(steer)
		out.fields["Drive Axles (lbs)"] = number(drive)
		out.fields["Trailer Axles (lbs)"] = number(trailer)
		out.fields["Gross Weight (lbs)"] = number(steer + drive + trailer)
		out.fields["Weighed At"] = map[string]any{"dateTimeValue": map[string]any{"dateTime": stamp}}
	}
	return out
}

func documentException(fraction float64) string {
	switch {
	case fraction < 0.85:
		return "None"
	case fraction < 0.9:
		return "Shortage"
	case fraction < 0.93:
		return "Overage"
	default:
		return "Damage"
	}
}

func fieldDefinition(docType Record, label string) Record {
	for _, raw := range listOf(docType["fieldTypes"]) {
		if def, ok := anyAsMap(raw); ok && stringOf(def["label"]) == label {
			return Record(def)
		}
	}
	return Record{}
}
