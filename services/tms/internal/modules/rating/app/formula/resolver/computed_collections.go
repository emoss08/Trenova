package resolver

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/shopspring/decimal"
)

func computeStops(entity any) (any, error) {
	ordered, _, err := stopsOf(entity)
	if err != nil {
		return []any{}, err
	}

	stops := make([]any, 0, len(ordered))
	for _, stop := range ordered {
		stops = append(stops, stopRecord(stop))
	}

	return stops, nil
}

func stopRecord(stop *shipment.Stop) map[string]any {
	return map[string]any{
		"type":            string(stop.Type),
		"status":          string(stop.Status),
		"scheduleType":    string(stop.ScheduleType),
		"sequence":        stop.Sequence,
		"pieces":          optionalInt64(stop.Pieces),
		"weight":          optionalInt64(stop.Weight),
		"windowStart":     positiveTimestamp(stop.ScheduledWindowStart),
		"windowEnd":       optionalTimestamp(stop.ScheduledWindowEnd),
		"actualArrival":   optionalTimestamp(stop.ActualArrival),
		"actualDeparture": optionalTimestamp(stop.ActualDeparture),
		"city":            stopCity(stop),
		"state":           stopState(stop),
		"zip":             stopZip(stop),
		"timezone":        stopTimezone(stop),
	}
}

func optionalInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalFeet(value *float64) any {
	if value == nil {
		return nil
	}
	return decimal.NewFromFloat(*value)
}

func positiveTimestamp(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func optionalTimestamp(value *int64) any {
	if value == nil {
		return nil
	}
	return positiveTimestamp(*value)
}

func computeCommodities(entity any) (any, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return []any{}, err
	}

	lines := make([]any, 0, len(entry.Commodities))
	for _, line := range entry.Commodities {
		if line != nil {
			lines = append(lines, commodityRecord(line))
		}
	}

	return lines, nil
}

func commodityRecord(line *shipment.ShipmentCommodity) map[string]any {
	record := map[string]any{
		"name":         "",
		"freightClass": "",
		"stackable":    false,
		"hazmat":       false,
		"weight":       line.Weight,
		"pieces":       line.Pieces,
		"lengthFeet":   optionalFeet(line.LengthFeet),
		"widthFeet":    optionalFeet(line.WidthFeet),
		"heightFeet":   optionalFeet(line.HeightFeet),
		"cubicFeet":    nil,
		"density":      nil,
	}

	if line.Commodity != nil {
		record["name"] = line.Commodity.Name
		record["freightClass"] = line.Commodity.FreightClass.Number()
		record["stackable"] = line.Commodity.Stackable
		record["hazmat"] = line.Commodity.HazardousMaterialID.IsNotNil() ||
			line.Commodity.HazardousMaterial != nil
	}

	if cube := line.CubicFeet(); cube.IsPositive() {
		record["cubicFeet"] = cube
		record["density"] = decimal.NewFromInt(line.Weight).Div(cube)
	}

	return record
}

func computeTotalCubicFeet(entity any) (any, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return decimal.Zero, err
	}
	return entry.RatingCubicFeet(), nil
}

func computeDensity(entity any) (any, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return nil, err
	}
	density, ok := entry.RatingDensity()
	if !ok {
		return nil, nil //nolint:nilnil // no cube means density is unknown
	}
	return density, nil
}

func computePrimaryFreightClass(entity any) (any, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return "", err
	}
	return entry.PrimaryFreightClass().Number(), nil
}

func computeHighestFreightClass(entity any) (any, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return "", err
	}
	return entry.HighestFreightClass().Number(), nil
}
