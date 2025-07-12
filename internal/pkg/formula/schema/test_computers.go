package schema

import (
	"strings"
)

// RegisterTestComputers registers compute functions that work with map data for testing
func RegisterTestComputers(resolver *DefaultDataResolver) {
	resolver.RegisterComputer("computeTemperatureDifferential", computeTemperatureDifferentialMap)
	resolver.RegisterComputer("computeHasHazmat", computeHasHazmatMap)
	resolver.RegisterComputer("computeRequiresTemperatureControl", computeRequiresTemperatureControlMap)
	resolver.RegisterComputer("computeTotalCommodityWeight", computeTotalCommodityWeightMap)
	resolver.RegisterComputer("computeIsExpedited", computeIsExpeditedMap)
	resolver.RegisterComputer("computeIsSameDay", computeIsSameDayMap)
	resolver.RegisterComputer("computeIsNextDay", computeIsNextDayMap)
	resolver.RegisterComputer("computeTotalDistance", computeTotalDistanceMap)
}

// computeTemperatureDifferentialMap calculates the difference between max and min temperature from map data
func computeTemperatureDifferentialMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return 0.0, nil
	}

	tempMax, hasMax := getFloat64(m, "temperatureMax")
	tempMin, hasMin := getFloat64(m, "temperatureMin")

	if hasMax && hasMin {
		return tempMax - tempMin, nil
	}
	return 0.0, nil
}

// computeHasHazmatMap checks if any commodity has hazardous material from map data
func computeHasHazmatMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return false, nil
	}

	commodities, ok := m["Commodities"].([]map[string]any)
	if !ok {
		// Try camelCase version
		commodities, ok = m["commodities"].([]map[string]any)
	}
	if !ok {
		// Try []any
		commoditiesAny, ok := m["Commodities"].([]any)
		if !ok {
			commoditiesAny, ok = m["commodities"].([]any)
		}
		if !ok {
			return false, nil
		}
		// Convert to []map[string]any
		commodities = make([]map[string]any, 0, len(commoditiesAny))
		for _, c := range commoditiesAny {
			if cm, ok := c.(map[string]any); ok {
				commodities = append(commodities, cm)
			}
		}
	}

	for _, commodity := range commodities {
		var commodityData map[string]any
		var ok bool
		
		// Try both case variations
		if commodityData, ok = commodity["Commodity"].(map[string]any); !ok {
			commodityData, ok = commodity["commodity"].(map[string]any)
		}
		
		if ok {
			// Try both case variations for hazardous material
			if _, hasHazmat := commodityData["HazardousMaterial"]; hasHazmat {
				return true, nil
			}
			if _, hasHazmat := commodityData["hazardousMaterial"]; hasHazmat {
				return true, nil
			}
		}
	}

	return false, nil
}

// computeRequiresTemperatureControlMap checks if shipment requires temperature control from map data
func computeRequiresTemperatureControlMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return false, nil
	}

	tempMax, hasMax := getFloat64(m, "temperatureMax")
	tempMin, hasMin := getFloat64(m, "temperatureMin")

	if hasMax && hasMin {
		// Requires control if range is significant or if temps are extreme
		diff := tempMax - tempMin
		return diff > 10 || tempMin < 32 || tempMax > 80, nil
	}

	return false, nil
}

// computeTotalCommodityWeightMap calculates total weight of all commodities from map data
func computeTotalCommodityWeightMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return 0.0, nil
	}

	commodities, ok := m["Commodities"].([]map[string]any)
	if !ok {
		// Try camelCase version
		commodities, ok = m["commodities"].([]map[string]any)
	}
	if !ok {
		// Try []any
		commoditiesAny, ok := m["Commodities"].([]any)
		if !ok {
			commoditiesAny, ok = m["commodities"].([]any)
		}
		if !ok {
			return 0.0, nil
		}
		// Convert to []map[string]any
		commodities = make([]map[string]any, 0, len(commoditiesAny))
		for _, c := range commoditiesAny {
			if cm, ok := c.(map[string]any); ok {
				commodities = append(commodities, cm)
			}
		}
	}

	var total float64
	for _, commodity := range commodities {
		// Try both case variations
		if weight, ok := getFloat64(commodity, "Weight"); ok {
			total += weight
		} else if weight, ok := getFloat64(commodity, "weight"); ok {
			total += weight
		}
	}

	return total, nil
}

// computeIsExpeditedMap checks if shipment is expedited service
func computeIsExpeditedMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return false, nil
	}

	// Check ServiceType first
	var serviceType map[string]any
	var stOk bool
	if serviceType, stOk = m["ServiceType"].(map[string]any); !stOk {
		serviceType, stOk = m["serviceType"].(map[string]any)
	}
	if stOk {
		if code, ok := serviceType["Code"].(string); ok {
			// Check if service type code indicates expedited service
			expeditedCodes := []string{"EXP", "EXPEDITED", "EXPRESS", "URGENT"}
			for _, expCode := range expeditedCodes {
				if code == expCode {
					return true, nil
				}
			}
		}
		if desc, ok := serviceType["Description"].(string); ok {
			// Check description for expedited keywords
			expDesc := strings.ToLower(desc)
			if strings.Contains(expDesc, "expedited") || strings.Contains(expDesc, "express") || strings.Contains(expDesc, "urgent") {
				return true, nil
			}
		}
	}

	// Check ShipmentType as fallback
	var shipmentType map[string]any
	var shOk bool
	if shipmentType, shOk = m["ShipmentType"].(map[string]any); !shOk {
		shipmentType, shOk = m["shipmentType"].(map[string]any)
	}
	if shOk {
		if code, ok := shipmentType["Code"].(string); ok {
			expeditedCodes := []string{"EXP", "EXPEDITED", "EXPRESS", "URGENT"}
			for _, expCode := range expeditedCodes {
				if code == expCode {
					return true, nil
				}
			}
		}
		if desc, ok := shipmentType["Description"].(string); ok {
			expDesc := strings.ToLower(desc)
			if strings.Contains(expDesc, "expedited") || strings.Contains(expDesc, "express") || strings.Contains(expDesc, "urgent") {
				return true, nil
			}
		}
	}

	return false, nil
}

// computeIsSameDayMap checks if shipment is same day service
func computeIsSameDayMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return false, nil
	}

	// Check ServiceType first
	var serviceType map[string]any
	var stOk bool
	if serviceType, stOk = m["ServiceType"].(map[string]any); !stOk {
		serviceType, stOk = m["serviceType"].(map[string]any)
	}
	if stOk {
		if code, ok := serviceType["Code"].(string); ok {
			sameDayCodes := []string{"SAME", "SAMEDAY", "SD", "TODAY"}
			for _, sdCode := range sameDayCodes {
				if code == sdCode {
					return true, nil
				}
			}
		}
		if desc, ok := serviceType["Description"].(string); ok {
			sdDesc := strings.ToLower(desc)
			if strings.Contains(sdDesc, "same day") || strings.Contains(sdDesc, "sameday") || strings.Contains(sdDesc, "today") {
				return true, nil
			}
		}
	}

	// Check ShipmentType as fallback
	var shipmentType map[string]any
	var shOk bool
	if shipmentType, shOk = m["ShipmentType"].(map[string]any); !shOk {
		shipmentType, shOk = m["shipmentType"].(map[string]any)
	}
	if shOk {
		if code, ok := shipmentType["Code"].(string); ok {
			sameDayCodes := []string{"SAME", "SAMEDAY", "SD", "TODAY"}
			for _, sdCode := range sameDayCodes {
				if code == sdCode {
					return true, nil
				}
			}
		}
		if desc, ok := shipmentType["Description"].(string); ok {
			sdDesc := strings.ToLower(desc)
			if strings.Contains(sdDesc, "same day") || strings.Contains(sdDesc, "sameday") || strings.Contains(sdDesc, "today") {
				return true, nil
			}
		}
	}

	return false, nil
}

// computeIsNextDayMap checks if shipment is next day service
func computeIsNextDayMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return false, nil
	}

	// Check ServiceType first
	var serviceType map[string]any
	var stOk bool
	if serviceType, stOk = m["ServiceType"].(map[string]any); !stOk {
		serviceType, stOk = m["serviceType"].(map[string]any)
	}
	if stOk {
		if code, ok := serviceType["Code"].(string); ok {
			nextDayCodes := []string{"NEXT", "NEXTDAY", "ND", "OVERNIGHT"}
			for _, ndCode := range nextDayCodes {
				if code == ndCode {
					return true, nil
				}
			}
		}
		if desc, ok := serviceType["Description"].(string); ok {
			ndDesc := strings.ToLower(desc)
			if strings.Contains(ndDesc, "next day") || strings.Contains(ndDesc, "nextday") || strings.Contains(ndDesc, "overnight") {
				return true, nil
			}
		}
	}

	// Check ShipmentType as fallback
	var shipmentType map[string]any
	var shOk bool
	if shipmentType, shOk = m["ShipmentType"].(map[string]any); !shOk {
		shipmentType, shOk = m["shipmentType"].(map[string]any)
	}
	if shOk {
		if code, ok := shipmentType["Code"].(string); ok {
			nextDayCodes := []string{"NEXT", "NEXTDAY", "ND", "OVERNIGHT"}
			for _, ndCode := range nextDayCodes {
				if code == ndCode {
					return true, nil
				}
			}
		}
		if desc, ok := shipmentType["Description"].(string); ok {
			ndDesc := strings.ToLower(desc)
			if strings.Contains(ndDesc, "next day") || strings.Contains(ndDesc, "nextday") || strings.Contains(ndDesc, "overnight") {
				return true, nil
			}
		}
	}

	return false, nil
}

// computeTotalDistanceMap calculates total distance from moves
func computeTotalDistanceMap(entity any) (any, error) {
	m, ok := entity.(map[string]any)
	if !ok {
		return 0.0, nil
	}

	// Get moves from the shipment
	moves, ok := m["Moves"].([]map[string]any)
	if !ok {
		// Try camelCase version
		moves, ok = m["moves"].([]map[string]any)
	}
	if !ok {
		// Try []any
		movesAny, ok := m["Moves"].([]any)
		if !ok {
			movesAny, ok = m["moves"].([]any)
		}
		if !ok {
			return 0.0, nil
		}
		// Convert to []map[string]any
		moves = make([]map[string]any, 0, len(movesAny))
		for _, move := range movesAny {
			if moveMap, ok := move.(map[string]any); ok {
				moves = append(moves, moveMap)
			}
		}
	}

	var totalDistance float64
	for _, move := range moves {
		// Try both case variations
		if distance, ok := getFloat64(move, "Distance"); ok {
			totalDistance += distance
		} else if distance, ok := getFloat64(move, "distance"); ok {
			totalDistance += distance
		}
	}

	// If no distance found in moves, return default for testing
	if totalDistance == 0 {
		return 500.0, nil // Default distance for testing
	}

	return totalDistance, nil
}

// getFloat64 safely extracts a float64 value from a map
func getFloat64(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}

	switch val := v.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case int16:
		return float64(val), true
	case int:
		return float64(val), true
	default:
		return 0, false
	}
}
