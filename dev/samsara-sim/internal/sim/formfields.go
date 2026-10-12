package sim

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	formFieldTypeDateTime  = "datetime"
	formFieldTypeAsset     = "asset"
	formFieldTypePerson    = "person"
	formFieldTypeTable     = "table"
	formFieldTypeGeofence  = "geofence"
	formFieldTypeBarcode   = "barcode"
	formFieldTypeMedia     = "media"
	formEntryTypeTracked   = "tracked"
	formUserTypeUser       = "user"
	formDateTimeTypeDate   = "date"
	formDateTimeTypeTime   = "time"
	formMediaMaxDecodedLen = 1536 * 1024
)

var (
	formInputFieldTypes = []string{
		formFieldTypeNumber, formFieldTypeText, formFieldTypeMultipleChoice,
		formFieldTypeCheckBoxes, formFieldTypeDateTime, formFieldTypeAsset,
		formFieldTypePerson, formFieldTypeTable, formFieldTypeGeofence,
		formFieldTypeBarcode, formFieldTypeMedia,
	}
	formTableCellTypes = []string{
		formFieldTypeNumber, formFieldTypeText, formFieldTypeMultipleChoice,
		formFieldTypeCheckBoxes, formFieldTypeDateTime, formFieldTypePerson,
		formFieldTypeBarcode,
	}
	formValueKeys = map[string]string{
		formFieldTypeNumber:         "numberValue",
		formFieldTypeText:           "textValue",
		formFieldTypeMultipleChoice: "multipleChoiceValue",
		formFieldTypeCheckBoxes:     "checkBoxesValue",
		formFieldTypeDateTime:       "dateTimeValue",
		formFieldTypeAsset:          "assetValue",
		formFieldTypePerson:         "personValue",
		formFieldTypeTable:          "tableValue",
		formFieldTypeGeofence:       "geofenceValue",
		formFieldTypeBarcode:        "barcodeValue",
		formFieldTypeMedia:          "mediaValue",
		formFieldTypeSignature:      "signatureValue",
	}
	formAssetTypeAliases = map[string]string{
		assetTypeVehicle: assetTypeVehicle,
		assetTypeTrailer: assetTypeTrailer,
		"equipment":      assetTypeEquipment,
		"unpoweredAsset": assetTypeUnpowered,
	}
	formMediaTypes = map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/gif":       true,
		"image/webp":      true,
		"application/pdf": true,
		"video/mp4":       false,
		"video/quicktime": false,
	}
)

type formInputContext struct {
	view      *fleetView
	template  Record
	fields    map[string]Record
	users     map[string]struct{}
	mediaSeed string
}

func newFormInputContext(
	view *fleetView,
	template Record,
	users map[string]struct{},
	mediaSeed string,
) *formInputContext {
	defs := listOf(template["fields"])
	fields := make(map[string]Record, len(defs))
	for _, raw := range defs {
		if def, ok := anyAsMap(raw); ok {
			fields[stringValue(Record(def), keyID)] = Record(def)
		}
	}
	return &formInputContext{
		view:      view,
		template:  template,
		fields:    fields,
		users:     users,
		mediaSeed: mediaSeed,
	}
}

func (c *formInputContext) sanitizeInputs(raw any) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField("fields", "must be an array")
	}
	out := make([]any, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for idx, item := range items {
		path := fmt.Sprintf("fields[%d]", idx)
		input, err := c.sanitizeInput(item, path)
		if err != nil {
			return nil, err
		}
		fieldID := stringOf(input[keyID])
		if _, dup := seen[fieldID]; dup {
			return nil, invalidField(path+".id", "appears more than once")
		}
		seen[fieldID] = struct{}{}
		out = append(out, input)
	}
	return out, nil
}

func (c *formInputContext) sanitizeInput(raw any, path string) (map[string]any, error) {
	input, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(path, "must be an object")
	}
	fieldID, err := requiredString(input, keyID, path)
	if err != nil {
		return nil, err
	}
	fieldType, err := requiredEnum(input, keyType, path, formInputFieldTypes)
	if err != nil {
		return nil, err
	}
	def, exists := c.fields[fieldID]
	if !exists {
		return nil, invalidField(
			path+".id",
			fmt.Sprintf("%q is not a field of form template %s", fieldID, recordID(c.template)),
		)
	}
	if defType := stringValue(def, keyType); defType != fieldType {
		return nil, invalidField(
			path+".type",
			fmt.Sprintf("is %q but field %s is a %q field", fieldType, fieldID, defType),
		)
	}
	value, err := exclusiveValue(input, fieldType, path)
	if err != nil {
		return nil, err
	}
	valuePath := path + "." + formValueKeys[fieldType]
	rendered, err := c.renderValue(def, fieldType, value, valuePath)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		keyID:                    fieldID,
		"label":                  stringValue(def, "label"),
		keyType:                  fieldType,
		formValueKeys[fieldType]: rendered,
	}, nil
}

func exclusiveValue(input map[string]any, fieldType, path string) (map[string]any, error) {
	expected := formValueKeys[fieldType]
	for candidateType, key := range formValueKeys {
		if key == expected {
			continue
		}
		if _, present := input[key]; present {
			return nil, invalidField(
				path+"."+key,
				fmt.Sprintf("is only valid for %s form input fields", candidateType),
			)
		}
	}
	value, ok := anyAsMap(input[expected])
	if !ok {
		return nil, invalidField(
			path+"."+expected,
			fmt.Sprintf("is required as an object for %s fields", fieldType),
		)
	}
	return value, nil
}

func (c *formInputContext) renderValue(
	def Record,
	fieldType string,
	value map[string]any,
	path string,
) (map[string]any, error) {
	switch fieldType {
	case formFieldTypeNumber:
		return renderFormNumber(def, value, path)
	case formFieldTypeText:
		text, err := requiredRawString(value, keyValue, path)
		if err != nil {
			return nil, err
		}
		return map[string]any{keyValue: text}, nil
	case formFieldTypeMultipleChoice:
		return renderFormChoice(def, value, path)
	case formFieldTypeCheckBoxes:
		return renderFormCheckBoxes(def, value, path)
	case formFieldTypeDateTime:
		return renderFormDateTime(def, value, path)
	case formFieldTypeAsset:
		return c.renderFormAsset(def, value, path)
	case formFieldTypePerson:
		return c.renderFormPerson(def, value, path)
	case formFieldTypeGeofence:
		return c.renderFormGeofence(value, path)
	case formFieldTypeBarcode:
		return renderFormBarcodes(value, path)
	case formFieldTypeMedia:
		return c.renderFormMedia(stringValue(def, keyID), value, path)
	case formFieldTypeTable:
		return c.renderFormTable(def, value, path)
	default:
		return nil, invalidField(path, "is not supported")
	}
}

func requiredString(input map[string]any, key, path string) (string, error) {
	text, err := requiredRawString(input, key, path)
	if err != nil {
		return "", err
	}
	clean := strings.TrimSpace(text)
	if clean == "" {
		return "", invalidField(path+"."+key, "must not be empty")
	}
	return clean, nil
}

func requiredRawString(input map[string]any, key, path string) (string, error) {
	raw, present := input[key]
	if !present || raw == nil {
		return "", invalidField(path+"."+key, "is required")
	}
	text, ok := raw.(string)
	if !ok {
		return "", invalidField(path+"."+key, "must be a string")
	}
	return text, nil
}

func requiredEnum(input map[string]any, key, path string, allowed []string) (string, error) {
	value, err := requiredString(input, key, path)
	if err != nil {
		return "", err
	}
	if !slices.Contains(allowed, value) {
		return "", invalidField(path+"."+key, "must be one of "+strings.Join(quoteAll(allowed), ", "))
	}
	return value, nil
}

func renderFormNumber(def Record, value map[string]any, path string) (map[string]any, error) {
	number, ok := value[keyValue].(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return nil, invalidField(path+".value", "must be a number")
	}
	if places, has := int64Value(def["numDecimalPlaces"]); has && decimalPlaces(number) > int(places) {
		return nil, invalidField(
			path+".value",
			fmt.Sprintf("allows at most %d decimal places", places),
		)
	}
	return map[string]any{keyValue: number}, nil
}

func decimalPlaces(value float64) int {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	_, fraction, found := strings.Cut(text, ".")
	if !found {
		return 0
	}
	return len(fraction)
}

func formOptions(def Record) map[string]string {
	options := listOf(def["options"])
	out := make(map[string]string, len(options))
	for _, raw := range options {
		if option, ok := anyAsMap(raw); ok {
			out[stringOf(option[keyID])] = stringOf(option["label"])
		}
	}
	return out
}

func renderFormChoice(def Record, value map[string]any, path string) (map[string]any, error) {
	valueID, err := requiredString(value, "valueId", path)
	if err != nil {
		return nil, err
	}
	label, ok := formOptions(def)[valueID]
	if !ok {
		return nil, invalidField(path+".valueId", fmt.Sprintf("%q is not an option of this field", valueID))
	}
	return map[string]any{keyValue: label, "valueId": valueID}, nil
}

func renderFormCheckBoxes(def Record, value map[string]any, path string) (map[string]any, error) {
	raw, ok := value["valueIds"].([]any)
	if !ok {
		return nil, invalidField(path+".valueIds", "is required as an array of option IDs")
	}
	options := formOptions(def)
	labels := make([]any, 0, len(raw))
	ids := make([]any, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for idx, item := range raw {
		optionID, isString := item.(string)
		label, known := options[optionID]
		if !isString || !known {
			return nil, invalidField(
				fmt.Sprintf("%s.valueIds[%d]", path, idx),
				"is not an option of this field",
			)
		}
		if _, dup := seen[optionID]; dup {
			continue
		}
		seen[optionID] = struct{}{}
		labels = append(labels, label)
		ids = append(ids, optionID)
	}
	return map[string]any{keyValue: labels, "valueIds": ids}, nil
}

func renderFormDateTime(def Record, value map[string]any, path string) (map[string]any, error) {
	text, err := requiredString(value, keyValue, path)
	if err != nil {
		return nil, err
	}
	parsed, err := parseRFC3339(text)
	if err != nil {
		return nil, invalidField(path+".value", "must be an RFC 3339 timestamp")
	}
	kind := stringValue(def, "allowedDateTimeValueType")
	if kind == "" {
		kind = formFieldTypeDateTime
	}
	out := map[string]any{keyType: kind, keyValue: parsed.Format(time.RFC3339)}
	if kind == formDateTimeTypeDate {
		out["dateValue"] = parsed.Format(dateLayout)
	}
	return out, nil
}

func (c *formInputContext) renderFormAsset(
	def Record,
	value map[string]any,
	path string,
) (map[string]any, error) {
	asset, ok := anyAsMap(value[keyAsset])
	if !ok {
		return nil, invalidField(path+".asset", "is required")
	}
	assetID, err := requiredString(asset, keyID, path+".asset")
	if err != nil {
		return nil, err
	}
	record, exists := c.view.snap.assetByID[assetID]
	if !exists {
		return nil, missingReference(keyAsset, assetID)
	}
	if allowed := stringListValues(def["allowedAssetTypes"]); len(allowed) > 0 {
		kind := assetType(record)
		permitted := false
		for _, alias := range allowed {
			if formAssetTypeAliases[alias] == kind {
				permitted = true
				break
			}
		}
		if !permitted {
			return nil, invalidField(
				path+".asset.id",
				fmt.Sprintf("is a %s; this field accepts %s", kind, strings.Join(allowed, ", ")),
			)
		}
	}
	return map[string]any{keyAsset: formTrackedAsset(record)}, nil
}

func formTrackedAsset(record Record) map[string]any {
	return map[string]any{
		"entryType":      formEntryTypeTracked,
		keyID:            recordID(record),
		fieldExternalIDs: renderExternalIDs(record, assetAutoExternalIDs),
	}
}

func parsePolymorphicUserID(raw string) (kind, id string, ok bool) {
	kind, id, found := strings.Cut(strings.TrimSpace(raw), "-")
	if !found || strings.TrimSpace(id) == "" {
		return "", "", false
	}
	switch kind {
	case formSubmitterTypeDriver, formUserTypeUser:
		return kind, strings.TrimSpace(id), true
	default:
		return "", "", false
	}
}

func (c *formInputContext) renderFormPerson(
	def Record,
	value map[string]any,
	path string,
) (map[string]any, error) {
	person, ok := anyAsMap(value["person"])
	if !ok {
		return nil, invalidField(path+".person", "is required")
	}
	raw, err := requiredString(person, "polymorphicUserId", path+".person")
	if err != nil {
		return nil, err
	}
	kind, id, valid := parsePolymorphicUserID(raw)
	if !valid {
		return nil, invalidField(
			path+".person.polymorphicUserId",
			"must be driver-<driverId> or user-<userId>",
		)
	}
	if kind == formSubmitterTypeDriver && def["includeDrivers"] == false ||
		kind == formUserTypeUser && def["includeUsers"] == false {
		return nil, invalidField(path+".person.polymorphicUserId", "this field does not accept "+kind+"s")
	}
	if err = c.ensurePolymorphicUser(kind, id); err != nil {
		return nil, err
	}
	return map[string]any{"person": map[string]any{
		"entryType":         formEntryTypeTracked,
		"polymorphicUserId": map[string]any{keyID: id, keyType: kind},
	}}, nil
}

func (c *formInputContext) ensurePolymorphicUser(kind, id string) error {
	if kind == formSubmitterTypeDriver {
		if _, ok := c.view.snap.driverByID[id]; !ok {
			return missingReference(keyDriver, id)
		}
		return nil
	}
	if _, ok := c.users[id]; !ok {
		return missingReference(formUserTypeUser, id)
	}
	return nil
}

func (c *formInputContext) renderFormGeofence(value map[string]any, path string) (map[string]any, error) {
	geofence, ok := anyAsMap(value["geofence"])
	if !ok {
		return nil, invalidField(path+".geofence", "is required")
	}
	addressID, err := requiredString(geofence, keyID, path+".geofence")
	if err != nil {
		return nil, err
	}
	address, exists := c.view.snap.addressByID[addressID]
	if !exists {
		return nil, missingReference("address", addressID)
	}
	return map[string]any{"geofence": formTrackedGeofence(address)}, nil
}

func formTrackedGeofence(address Record) map[string]any {
	return map[string]any{
		"entryType":      formEntryTypeTracked,
		keyID:            recordID(address),
		"address":        stringValue(address, "formattedAddress"),
		fieldExternalIDs: renderExternalIDs(address, nil),
	}
}

func renderFormBarcodes(value map[string]any, path string) (map[string]any, error) {
	raw, ok := value["barcodes"].([]any)
	if !ok {
		return nil, invalidField(path+".barcodes", "is required as an array")
	}
	out := make([]any, 0, len(raw))
	for idx, item := range raw {
		entry, isMap := anyAsMap(item)
		itemPath := fmt.Sprintf("%s.barcodes[%d]", path, idx)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		text, err := requiredString(entry, keyValue, itemPath)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{keyValue: text})
	}
	return map[string]any{"barcodes": out}, nil
}

func (c *formInputContext) renderFormMedia(
	fieldID string,
	value map[string]any,
	path string,
) (map[string]any, error) {
	raw, ok := value["mediaList"].([]any)
	if !ok || len(raw) == 0 {
		return nil, invalidField(path+".mediaList", "is required as a non-empty array")
	}
	out := make([]any, 0, len(raw))
	for idx, item := range raw {
		itemPath := fmt.Sprintf("%s.mediaList[%d]", path, idx)
		entry, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(itemPath, "must be an object")
		}
		mediaType, err := requiredString(entry, "mediaType", itemPath)
		if err != nil {
			return nil, err
		}
		payload, err := requiredString(entry, "base64Payload", itemPath)
		if err != nil {
			return nil, err
		}
		if err = validateFormMediaPayload(mediaType, payload, itemPath); err != nil {
			return nil, err
		}
		mediaID := deterministicUUID(
			c.mediaSeed,
			fieldID,
			strconv.Itoa(idx),
			strconv.FormatUint(fnvHash64(payload), 16),
		)
		out = append(out, formMediaRecord(mediaID))
	}
	return map[string]any{"mediaList": out}, nil
}

func validateFormMediaPayload(mediaType, payload, path string) error {
	sniffed, known := formMediaTypes[mediaType]
	if !known {
		allowed := make([]string, 0, len(formMediaTypes))
		for key := range formMediaTypes {
			allowed = append(allowed, key)
		}
		slices.Sort(allowed)
		return invalidField(path+".mediaType", "must be one of "+strings.Join(quoteAll(allowed), ", "))
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(decoded) == 0 {
		return invalidField(path+".base64Payload", "must be non-empty standard base64")
	}
	if len(decoded) > formMediaMaxDecodedLen {
		return invalidField(path+".base64Payload", "exceeds the 1.5 MB media limit")
	}
	if sniffed && http.DetectContentType(decoded) != mediaType {
		return invalidField(path+".base64Payload", "does not contain "+mediaType+" content")
	}
	return nil
}

func formMediaRecord(mediaID string) map[string]any {
	return map[string]any{
		keyID:              mediaID,
		"processingStatus": formMediaProcessingStatusFinished,
		"url":              formMediaURLPrefix + mediaID,
	}
}

func (c *formInputContext) renderFormTable(
	def Record,
	value map[string]any,
	path string,
) (map[string]any, error) {
	columnDefs := listOf(def["columns"])
	columns := make([]any, 0, len(columnDefs))
	byID := make(map[string]Record, len(columnDefs))
	for _, raw := range columnDefs {
		column, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		byID[stringOf(column[keyID])] = Record(column)
		columns = append(columns, map[string]any{
			keyID:   column[keyID],
			"label": column["label"],
			keyType: column[keyType],
		})
	}
	rawRows, ok := value["rows"].([]any)
	if !ok {
		return nil, invalidField(path+".rows", "is required as an array")
	}
	rows := make([]any, 0, len(rawRows))
	for idx, rawRow := range rawRows {
		rowPath := fmt.Sprintf("%s.rows[%d]", path, idx)
		row, isMap := anyAsMap(rawRow)
		if !isMap {
			return nil, invalidField(rowPath, "must be an object")
		}
		rowID, err := requiredString(row, keyID, rowPath)
		if err != nil {
			return nil, err
		}
		if !isUUID(rowID) {
			return nil, invalidField(rowPath+".id", "must be a UUID")
		}
		cells, err := c.renderTableCells(byID, row["cells"], rowPath+".cells")
		if err != nil {
			return nil, err
		}
		rows = append(rows, map[string]any{keyID: rowID, "cells": cells})
	}
	return map[string]any{"columns": columns, "rows": rows}, nil
}

func (c *formInputContext) renderTableCells(
	columns map[string]Record,
	raw any,
	path string,
) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "is required as an array")
	}
	out := make([]any, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for idx, item := range items {
		cellPath := fmt.Sprintf("%s[%d]", path, idx)
		cell, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(cellPath, "must be an object")
		}
		columnID, err := requiredString(cell, keyID, cellPath)
		if err != nil {
			return nil, err
		}
		cellType, err := requiredEnum(cell, keyType, cellPath, formTableCellTypes)
		if err != nil {
			return nil, err
		}
		column, exists := columns[columnID]
		if !exists {
			return nil, invalidField(cellPath+".id", fmt.Sprintf("%q is not a column of this table", columnID))
		}
		if columnType := stringValue(column, keyType); columnType != cellType {
			return nil, invalidField(cellPath+".type", fmt.Sprintf("must be %q for this column", columnType))
		}
		if _, dup := seen[columnID]; dup {
			return nil, invalidField(cellPath+".id", "appears more than once in the row")
		}
		seen[columnID] = struct{}{}
		value, err := exclusiveValue(cell, cellType, cellPath)
		if err != nil {
			return nil, err
		}
		valueKey := formValueKeys[cellType]
		rendered, err := c.renderValue(column, cellType, value, cellPath+"."+valueKey)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{keyID: columnID, keyType: cellType, valueKey: rendered})
	}
	return out, nil
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for idx := 0; idx < len(value); idx++ {
		char := value[idx]
		switch idx {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
				return false
			}
		}
	}
	return true
}

func mergeFormInputs(existing []any, updates []any) []any {
	out := make([]any, 0, len(existing)+len(updates))
	index := make(map[string]int, len(existing))
	for _, raw := range existing {
		input, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		index[stringOf(input[keyID])] = len(out)
		out = append(out, cloneAny(raw))
	}
	for _, raw := range updates {
		input, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		fieldID := stringOf(input[keyID])
		if position, exists := index[fieldID]; exists {
			out[position] = cloneAny(raw)
			continue
		}
		index[fieldID] = len(out)
		out = append(out, cloneAny(raw))
	}
	return out
}

func refreshFormMedia(raw any, expiresAt string) {
	switch typed := raw.(type) {
	case map[string]any:
		if typed["processingStatus"] == formMediaProcessingStatusFinished {
			if _, hasURL := typed["url"]; hasURL {
				typed["urlExpiresAt"] = expiresAt
			}
		}
		for _, value := range typed {
			refreshFormMedia(value, expiresAt)
		}
	case Record:
		refreshFormMedia(map[string]any(typed), expiresAt)
	case []any:
		for _, value := range typed {
			refreshFormMedia(value, expiresAt)
		}
	}
}
