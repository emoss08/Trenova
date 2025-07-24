/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package osm

import (
	"context"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
)

// EnhancedRestrictions contains comprehensive truck restrictions
type EnhancedRestrictions struct {
	MaxHeight        float64 // meters
	MaxWidth         float64 // meters
	MaxLength        float64 // meters
	MaxWeight        float64 // kg
	MaxAxleLoad      float64 // kg per axle
	HazmatAllowed    bool
	TruckAllowed     bool
	TollRoad         bool
	TruckSpeedLimit  int     // km/h
	BridgeMaxWeight  float64 // kg
	TunnelMaxHeight  float64 // meters
	TimeRestrictions string  // JSON of time-based restrictions
}

// unitRegex matches common unit patterns
var (
	// Height patterns: 13'6", 4.1m, 4.1 m, 13ft6in, etc.
	heightFeetInchesRegex = regexp.MustCompile(`(\d+)'(?:\s*)?(\d+)"?`)
	heightFeetRegex       = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:ft|feet)`)
	heightMetersRegex     = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*m(?:eters?)?`)

	// Weight patterns: 80000 lbs, 36t, 36 t, 36 tons, etc.
	weightPoundsRegex = regexp.MustCompile(`(\d+)\s*(?:lbs?|pounds?)`)
	weightTonsRegex   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:t|tons?)`)
	weightKgRegex     = regexp.MustCompile(`(\d+)\s*kg`)

	// Width/Length patterns: 102in, 2.6m, 8'6", etc.
	widthInchesRegex = regexp.MustCompile(`(\d+)\s*(?:in|inches)`)
	widthMetersRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*m(?:eters?)?`)
)

// ExtractEnhancedRestrictions extracts comprehensive truck restrictions from OSM tags
func ExtractEnhancedRestrictions(way *osm.Way) EnhancedRestrictions {
	r := EnhancedRestrictions{
		MaxHeight:       0,
		MaxWidth:        0,
		MaxLength:       0,
		MaxWeight:       0,
		MaxAxleLoad:     0,
		HazmatAllowed:   true,
		TruckAllowed:    true,
		TollRoad:        false,
		TruckSpeedLimit: 0,
		BridgeMaxWeight: 0,
		TunnelMaxHeight: 0,
	}

	// _ Parse height restrictions
	if maxHeight := way.Tags.Find("maxheight"); maxHeight != "" {
		r.MaxHeight = parseHeight(maxHeight)
	}
	if tunnelHeight := way.Tags.Find("tunnel:maxheight"); tunnelHeight != "" {
		r.TunnelMaxHeight = parseHeight(tunnelHeight)
	}

	// _ Parse weight restrictions
	if maxWeight := way.Tags.Find("maxweight"); maxWeight != "" {
		r.MaxWeight = parseWeight(maxWeight)
	}
	if bridgeWeight := way.Tags.Find("bridge:maxweight"); bridgeWeight != "" {
		r.BridgeMaxWeight = parseWeight(bridgeWeight)
	}
	if maxAxleLoad := way.Tags.Find("maxaxleload"); maxAxleLoad != "" {
		r.MaxAxleLoad = parseWeight(maxAxleLoad)
	}

	// _ Parse width restrictions
	if maxWidth := way.Tags.Find("maxwidth"); maxWidth != "" {
		r.MaxWidth = parseWidth(maxWidth)
	}

	// _ Parse length restrictions
	if maxLength := way.Tags.Find("maxlength"); maxLength != "" {
		r.MaxLength = parseLength(maxLength)
	}

	// _ Check truck access
	hgv := way.Tags.Find("hgv")
	if hgv == "no" || hgv == "destination" || hgv == "delivery" {
		r.TruckAllowed = false
	}

	// _ Also check access:hgv tag
	accessHGV := way.Tags.Find("access:hgv")
	if accessHGV == "no" {
		r.TruckAllowed = false
	}

	// _ Check hazmat restrictions
	hazmat := way.Tags.Find("hazmat")
	if hazmat == "no" {
		r.HazmatAllowed = false
	}

	// _ Check for toll roads
	toll := way.Tags.Find("toll")
	if toll == "yes" {
		r.TollRoad = true
	}

	// _ Get truck-specific speed limit
	if truckSpeed := way.Tags.Find("maxspeed:hgv"); truckSpeed != "" {
		r.TruckSpeedLimit = parseSpeed(truckSpeed)
	}

	// _ Check for time-based restrictions
	if hgvConditional := way.Tags.Find("hgv:conditional"); hgvConditional != "" {
		r.TimeRestrictions = hgvConditional
	}

	return r
}

// parseHeight converts various height formats to meters
func parseHeight(height string) float64 {
	height = strings.TrimSpace(height)

	// _ Try feet and inches format (e.g., "13'6"")
	if match := heightFeetInchesRegex.FindStringSubmatch(height); match != nil {
		feet, _ := strconv.ParseFloat(match[1], 64)
		inches, _ := strconv.ParseFloat(match[2], 64)
		return (feet*12 + inches) * 0.0254 // Convert to meters
	}

	// _ Try feet only format
	if match := heightFeetRegex.FindStringSubmatch(height); match != nil {
		feet, _ := strconv.ParseFloat(match[1], 64)
		return feet * 0.3048 // Convert to meters
	}

	// _ Try meters format
	if match := heightMetersRegex.FindStringSubmatch(height); match != nil {
		meters, _ := strconv.ParseFloat(match[1], 64)
		return meters
	}

	// _ Try simple number (assume meters)
	if val, err := strconv.ParseFloat(height, 64); err == nil {
		return val
	}

	return 0
}

// parseWeight converts various weight formats to kilograms
func parseWeight(weight string) float64 {
	weight = strings.TrimSpace(weight)

	// _ Try pounds format
	if match := weightPoundsRegex.FindStringSubmatch(weight); match != nil {
		pounds, _ := strconv.ParseFloat(match[1], 64)
		return pounds * 0.453592 // Convert to kg
	}

	// _ Try tons format
	if match := weightTonsRegex.FindStringSubmatch(weight); match != nil {
		tons, _ := strconv.ParseFloat(match[1], 64)
		return tons * 1000 // Convert to kg
	}

	// _ Try kg format
	if match := weightKgRegex.FindStringSubmatch(weight); match != nil {
		kg, _ := strconv.ParseFloat(match[1], 64)
		return kg
	}

	// _ Try simple number (assume tons if > 100, otherwise kg)
	if val, err := strconv.ParseFloat(weight, 64); err == nil {
		if val > 100 {
			return val // Assume kg
		}
		return val * 1000 // Assume tons
	}

	return 0
}

// parseWidth converts various width formats to meters
func parseWidth(width string) float64 {
	width = strings.TrimSpace(width)

	// _ Try inches format
	if match := widthInchesRegex.FindStringSubmatch(width); match != nil {
		inches, _ := strconv.ParseFloat(match[1], 64)
		return inches * 0.0254 // Convert to meters
	}

	// _ Try meters format
	if match := widthMetersRegex.FindStringSubmatch(width); match != nil {
		meters, _ := strconv.ParseFloat(match[1], 64)
		return meters
	}

	// _ Try simple number (assume meters)
	if val, err := strconv.ParseFloat(width, 64); err == nil {
		return val
	}

	return 0
}

// parseLength is an alias for parseWidth as they use the same units
func parseLength(length string) float64 {
	return parseWidth(length)
}

// parseSpeed converts speed strings to km/h
func parseSpeed(speed string) int {
	speed = strings.TrimSpace(speed)

	// _ Remove common suffixes
	speed = strings.TrimSuffix(speed, " mph")
	speed = strings.TrimSuffix(speed, " km/h")
	speed = strings.TrimSuffix(speed, " kph")

	if val, err := strconv.Atoi(speed); err == nil {
		// _ Check if it's likely mph (US speeds are typically 25, 35, 45, 55, 65, 70, 75)
		if val == 25 || val == 35 || val == 45 || val == 55 || val == 65 || val == 70 || val == 75 {
			return int(float64(val) * 1.60934) // Convert mph to km/h
		}
		return val // Assume km/h
	}

	return 0
}

// IsEnhancedDriveableRoad checks if a way is a driveable road with more categories
func IsEnhancedDriveableRoad(way *osm.Way) bool {
	highway := way.Tags.Find("highway")
	if highway == "" {
		return false
	}

	// _ Include more road types for comprehensive US coverage
	driveableTypes := map[string]bool{
		"motorway":          true,
		"trunk":             true,
		"primary":           true,
		"secondary":         true,
		"tertiary":          true,
		"unclassified":      true,
		"residential":       true,
		"motorway_link":     true,
		"trunk_link":        true,
		"primary_link":      true,
		"secondary_link":    true,
		"tertiary_link":     true,
		"living_street":     true, // Low-speed residential
		"service":           true, // Access roads, driveways
		"motorway_junction": true, // Highway ramps
	}

	// _ Exclude certain service roads
	if highway == "service" {
		service := way.Tags.Find("service")
		excludedServices := map[string]bool{
			"parking_aisle":    true,
			"driveway":         false, // Include driveways for last-mile routing
			"emergency_access": true,
		}
		if excluded, exists := excludedServices[service]; exists && excluded {
			return false
		}
	}

	// _ Check if explicitly marked as not for motor vehicles
	if motor := way.Tags.Find("motor_vehicle"); motor == "no" {
		return false
	}

	// _ Check access tag
	access := way.Tags.Find("access")
	if access == "no" || access == "private" {
		// _ But allow if explicitly marked for delivery/destination
		if hgv := way.Tags.Find("hgv"); hgv == "destination" || hgv == "delivery" {
			return true
		}
		return false
	}

	return driveableTypes[highway]
}

// BoundingBox represents a geographic bounding box
type BoundingBox struct {
	MinLat, MaxLat float64
	MinLon, MaxLon float64
}

// Contains checks if a point is within the bounding box
func (b BoundingBox) Contains(lat, lon float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lon >= b.MinLon && lon <= b.MaxLon
}

// USRegions defines bounding boxes for US regions
var USRegions = map[string]BoundingBox{
	"continental_us": {MinLat: 24.5, MaxLat: 49.4, MinLon: -125.0, MaxLon: -66.9},
	"northeast":      {MinLat: 38.0, MaxLat: 47.5, MinLon: -80.5, MaxLon: -66.9},
	"southeast":      {MinLat: 24.5, MaxLat: 38.0, MinLon: -91.7, MaxLon: -75.0},
	"midwest":        {MinLat: 36.0, MaxLat: 49.4, MinLon: -104.0, MaxLon: -80.5},
	"southwest":      {MinLat: 28.0, MaxLat: 37.0, MinLon: -117.2, MaxLon: -93.5},
	"west":           {MinLat: 32.5, MaxLat: 49.0, MinLon: -125.0, MaxLon: -102.0},
	"alaska":         {MinLat: 51.2, MaxLat: 71.4, MinLon: -180.0, MaxLon: -129.0},
	"hawaii":         {MinLat: 18.9, MaxLat: 22.3, MinLon: -160.3, MaxLon: -154.8},
}

// GetSpeedForRoadType returns estimated speed in km/h for different road types
func GetSpeedForRoadType(roadType string, isTruck bool) float64 {
	// _ Base speeds for cars
	speeds := map[string]float64{
		"motorway":       110, // ~70 mph
		"trunk":          90,  // ~55 mph
		"primary":        80,  // ~50 mph
		"secondary":      70,  // ~45 mph
		"tertiary":       60,  // ~35 mph
		"unclassified":   50,  // ~30 mph
		"residential":    40,  // ~25 mph
		"living_street":  20,  // ~12 mph
		"service":        30,  // ~20 mph
		"motorway_link":  70,  // ~45 mph
		"trunk_link":     60,  // ~35 mph
		"primary_link":   50,  // ~30 mph
		"secondary_link": 50,  // ~30 mph
		"tertiary_link":  40,  // ~25 mph
	}

	speed, exists := speeds[roadType]
	if !exists {
		speed = 50 // Default ~30 mph
	}

	// _ Reduce speeds for trucks
	if isTruck {
		switch roadType {
		case "motorway":
			return speed * 0.9 // Trucks slightly slower on highways
		case "residential", "living_street":
			return speed * 0.8 // Trucks much slower in residential areas
		default:
			return speed * 0.85 // General truck speed reduction
		}
	}

	return speed
}

// LogImportStats logs detailed import statistics
func LogImportStats(
	_ context.Context,
	nodeCount, wayCount, edgeCount int64,
	restrictions EnhancedRestrictions,
) {
	log.Printf("Import Statistics:")
	log.Printf("  - Nodes: %d", nodeCount)
	log.Printf("  - Ways: %d", wayCount)
	log.Printf("  - Edges: %d", edgeCount)
	log.Printf("  - Roads with height restrictions: %d", restrictions.MaxHeight)
	log.Printf("  - Roads with weight restrictions: %d", restrictions.MaxWeight)
	log.Printf("  - Toll roads: %d", restrictions.TollRoad)
	log.Printf("  - Roads restricted for trucks: %d", !restrictions.TruckAllowed)
}
