package resolver

import (
	"fmt"
	"reflect"
)

// computeStops lists every stop on the shipment, in move and sequence order,
// as plain records a formula can map, filter, and sum over. Each record
// carries the stop's own facts and its location's city, state, ZIP, and
// timezone, so a formula can count deliveries in a state or find the latest
// appointment window without reaching into nested structs.
func computeStops(entity any) (any, error) {
	stops := make([]any, 0, 4)

	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return stops, err
	}

	for _, move := range moves {
		moveStops, stopsErr := getFieldSlice(move, "Stops")
		if stopsErr != nil {
			continue
		}
		for _, stop := range moveStops {
			if stop == nil || isNilInterface(stop) {
				continue
			}
			stops = append(stops, stopRecord(stop))
		}
	}

	return stops, nil
}

func stopRecord(stop any) map[string]any {
	sequence, _ := getFieldInt64(stop, "Sequence")

	return map[string]any{
		"type":            fieldText(stop, "Type"),
		"status":          fieldText(stop, "Status"),
		"scheduleType":    fieldText(stop, "ScheduleType"),
		"sequence":        int(sequence),
		"pieces":          optionalNumber(stop, "Pieces"),
		"weight":          optionalNumber(stop, "Weight"),
		"windowStart":     optionalTimestamp(stop, "ScheduledWindowStart"),
		"windowEnd":       optionalTimestamp(stop, "ScheduledWindowEnd"),
		"actualArrival":   optionalTimestamp(stop, "ActualArrival"),
		"actualDeparture": optionalTimestamp(stop, "ActualDeparture"),
		"city":            stopLocationString(stop, "City"),
		"state":           stopLocationState(stop),
		"zip":             stopLocationString(stop, "PostalCode"),
		"timezone":        stopLocationString(stop, "Timezone"),
	}
}

// computeCommodities lists every commodity line with its class, dimensions,
// and hazmat flag. Cube and density are derived here once so dimensional
// weight formulas read them instead of re-multiplying per line.
func computeCommodities(entity any) (any, error) {
	lines := make([]any, 0, 4)

	commodities, err := getFieldSlice(entity, "Commodities")
	if err != nil {
		return lines, err
	}

	for _, line := range commodities {
		if line == nil || isNilInterface(line) {
			continue
		}
		lines = append(lines, commodityRecord(line))
	}

	return lines, nil
}

func commodityRecord(line any) map[string]any {
	weight, _ := getFieldFloat64(line, "Weight")
	pieces, _ := getFieldFloat64(line, "Pieces")

	record := map[string]any{
		"name":         "",
		"freightClass": "",
		"stackable":    false,
		"hazmat":       false,
		"weight":       weight,
		"pieces":       pieces,
		"lengthFeet":   optionalNumber(line, "LengthFeet"),
		"widthFeet":    optionalNumber(line, "WidthFeet"),
		"heightFeet":   optionalNumber(line, "HeightFeet"),
		"cubicFeet":    nil,
		"density":      nil,
	}

	if commodity, commErr := getFieldValue(line, "Commodity"); commErr == nil &&
		commodity != nil && !isNilInterface(commodity) {
		record["name"] = fieldText(commodity, "Name")
		record["freightClass"] = fieldText(commodity, "FreightClass")
		record["stackable"] = fieldBool(commodity, "Stackable")
		if hazmat, hazErr := getFieldValue(commodity, "HazardousMaterial"); hazErr == nil &&
			hazmat != nil && !isNilInterface(hazmat) {
			record["hazmat"] = true
		}
	}

	length, lengthOK := record["lengthFeet"].(float64)
	width, widthOK := record["widthFeet"].(float64)
	height, heightOK := record["heightFeet"].(float64)
	if lengthOK && widthOK && heightOK && length > 0 && width > 0 && height > 0 {
		perPiece := length * width * height
		count := pieces
		if count <= 0 {
			count = 1
		}
		cube := perPiece * count
		record["cubicFeet"] = cube
		if cube > 0 {
			record["density"] = weight / cube
		}
	}

	return record
}

// fieldText reads a string-kinded field, including named enum types, as text.
func fieldText(entity any, fieldName string) string {
	value, err := getFieldValue(entity, fieldName)
	if err != nil || value == nil {
		return ""
	}
	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return ""
}

func fieldBool(entity any, fieldName string) bool {
	value, err := getFieldValue(entity, fieldName)
	if err != nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case *bool:
		return typed != nil && *typed
	default:
		return false
	}
}

// optionalNumber reads a numeric field as float64, or nil when the field is
// an unset pointer, so a formula sees "no weight" rather than zero pounds.
func optionalNumber(entity any, fieldName string) any {
	value, err := getFieldValue(entity, fieldName)
	if err != nil || value == nil || isNilInterface(value) {
		return nil
	}
	if number, convErr := getFieldFloat64(entity, fieldName); convErr == nil {
		return number
	}
	integer, intErr := getFieldInt64(entity, fieldName)
	if intErr != nil {
		return nil
	}
	return float64(integer)
}

// optionalTimestamp reads a unix-seconds field, or nil when it is unset or
// zero, since no stop has ever happened at the epoch.
func optionalTimestamp(entity any, fieldName string) any {
	timestamp, err := getFieldInt64(entity, fieldName)
	if err != nil || timestamp <= 0 {
		return nil
	}
	return timestamp
}
