package pcmiler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	defaultStopCountry     = "US"
	stateReportTotalLine   = "TOTAL"
	mileageReportTypeValue = "MileageReportType:http://pcmiler.alk.com/APIs/v1.0"
	stateReportTypeValue   = "StateReportType:http://pcmiler.alk.com/APIs/v1.0"
)

type routeReport struct {
	Type             string            `json:"__type"`
	RouteID          string            `json:"RouteID"`
	ReportLines      []reportLine      `json:"ReportLines"`
	StateReportLines []stateReportLine `json:"StateReportLines"`
}

type reportLine struct {
	TMiles string `json:"TMiles"`
	LMiles string `json:"LMiles"`
	Dist   string `json:"Dist"`
	Warn   string `json:"Warn"`
}

type stateReportLine struct {
	StCntry string    `json:"StCntry"`
	Total   flexFloat `json:"Total"`
	Toll    flexFloat `json:"Toll"`
	Free    flexFloat `json:"Free"`
	Ferry   flexFloat `json:"Ferry"`
	Loaded  flexFloat `json:"Loaded"`
	Empty   flexFloat `json:"Empty"`
}

type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*f = 0
		return nil
	}
	if strings.HasPrefix(text, "\"") {
		var quoted string
		if err := sonic.Unmarshal(data, &quoted); err != nil {
			return fmt.Errorf("parse PC*Miler state report value %s: %w", text, err)
		}
		text = strings.ReplaceAll(strings.TrimSpace(quoted), ",", "")
		if text == "" {
			*f = 0
			return nil
		}
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return fmt.Errorf("parse PC*Miler state report value %s: %w", text, err)
	}
	*f = flexFloat(parsed)
	return nil
}

func buildRouteReportsPayload(routes []RouteRequest) map[string]any {
	payloadRoutes := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		stops := make([]map[string]any, 0, len(route.Stops))
		for idx, stop := range route.Stops {
			stops = append(stops, buildStop(stop, route.Options.Region, idx, len(route.Stops)))
		}

		payloadRoutes = append(payloadRoutes, map[string]any{
			"RouteId":          route.RouteID,
			"Stops":            stops,
			"RouteOptions":     buildOptions(route.Options),
			"VehicleType":      vehicleTypeCode(route.Options.VehicleType),
			"RoutingType":      routingTypeCode(route.Options.RoutingType),
			"ReportingOptions": buildReportingOptions(route.Options),
			"ReportTypes":      buildReportTypes(route.Options),
		})
	}

	return map[string]any{
		"ReportRoutes": payloadRoutes,
	}
}

func buildReportTypes(opts RouteOptions) []map[string]any {
	size := 1
	if opts.StateReport {
		size = 2
	}
	reportTypes := make([]map[string]any, 0, size)
	reportTypes = append(reportTypes, map[string]any{
		"__type":        mileageReportTypeValue,
		"TimeInSeconds": false,
	})
	if opts.StateReport {
		reportTypes = append(reportTypes, map[string]any{
			"__type": stateReportTypeValue,
		})
	}
	return reportTypes
}

func buildStop(stop Stop, region string, index int, totalStops int) map[string]any {
	label := fmt.Sprintf("Stop %d", index)
	if index == 0 {
		label = "Origin"
	} else if index == totalStops-1 {
		label = "Destination"
	}

	result := map[string]any{
		"Region": regionCode(region),
		"Label":  label,
		"ID":     label,
	}

	if stop.TrimblePlaceID != "" {
		result["PlaceId"] = stop.TrimblePlaceID
		return result
	}
	if stop.Latitude != nil && stop.Longitude != nil {
		result["Coords"] = map[string]any{
			"Lat": *stop.Latitude,
			"Lon": *stop.Longitude,
		}
		return result
	}

	address := map[string]any{
		"Country": stopCountry(stop.Country),
	}
	if stop.AddressLine != "" {
		address["StreetAddress"] = stop.AddressLine
	}
	if stop.City != "" {
		address["City"] = stop.City
	}
	if stop.State != "" {
		address["State"] = stop.State
	}
	if stop.PostalCode != "" {
		address["Zip"] = stop.PostalCode
	}
	result["Address"] = address
	return result
}

func buildOptions(opts RouteOptions) map[string]any {
	values := map[string]any{}
	setString(values, "ProfileName", opts.ProfileName)
	if opts.HighwayOnly {
		values["HighwayOnly"] = true
	}
	values["TollRoads"] = tollRoadsCode(opts.TollRoads)
	values["BordersOpen"] = opts.BordersOpen
	if opts.VehicleHeight > 0 {
		values["Height"] = opts.VehicleHeight
	}
	if opts.VehicleLength > 0 {
		values["Length"] = opts.VehicleLength
	}
	if opts.VehicleWidth > 0 {
		values["Width"] = opts.VehicleWidth
	}
	if opts.VehicleWeight > 0 {
		values["Weight"] = opts.VehicleWeight
	}
	if opts.Axles > 0 {
		values["Axles"] = opts.Axles
	}
	if len(opts.Hazmat) > 0 {
		values["HazMatTypes"] = hazmatCodes(opts.Hazmat)
	}
	values["DistanceUnits"] = distanceUnitsCode(opts.DistanceUnits)
	return values
}

func buildReportingOptions(opts RouteOptions) map[string]any {
	return map[string]any{
		"UseTollData": opts.IncludeTollData,
	}
}

func stopCountry(value string) string {
	country := strings.ToUpper(strings.TrimSpace(value))
	if country == "" {
		return defaultStopCountry
	}
	return country
}

func parseMileageResponse(reports []routeReport) []RouteMileage {
	results := make([]RouteMileage, 0, len(reports))
	indexByRoute := make(map[string]int, len(reports))
	for _, report := range reports {
		isMileage := strings.Contains(report.Type, "MileageReport")
		isState := strings.Contains(report.Type, "StateReport")
		if !isMileage && !isState {
			continue
		}
		idx, ok := indexByRoute[report.RouteID]
		if !ok {
			idx = len(results)
			indexByRoute[report.RouteID] = idx
			results = append(results, RouteMileage{RouteID: report.RouteID})
		}
		if isMileage {
			results[idx].Distance, results[idx].Warnings = mileageFromLines(report.ReportLines)
		}
		if isState {
			results[idx].JurisdictionDistances = jurisdictionsFromLines(report.StateReportLines)
		}
	}
	return results
}

func jurisdictionsFromLines(lines []stateReportLine) []JurisdictionDistance {
	jurisdictions := make([]JurisdictionDistance, 0, len(lines))
	for _, line := range lines {
		country, code, ok := parseStCntry(line.StCntry)
		if !ok {
			continue
		}
		jurisdictions = append(jurisdictions, JurisdictionDistance{
			Country:  country,
			Code:     code,
			Distance: float64(line.Total),
			Toll:     float64(line.Toll),
			Ferry:    float64(line.Ferry),
		})
	}
	return jurisdictions
}

func parseStCntry(value string) (string, string, bool) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" || trimmed == stateReportTotalLine {
		return "", "", false
	}
	country, code, found := strings.Cut(trimmed, "_")
	if !found {
		return defaultStopCountry, trimmed, true
	}
	country = strings.TrimSpace(country)
	code = strings.TrimSpace(code)
	if country == "" || code == "" {
		return "", "", false
	}
	return country, code, true
}

func mileageFromLines(lines []reportLine) (float64, []string) {
	warnings := make([]string, 0)
	var distance float64
	for _, line := range lines {
		if strings.TrimSpace(line.Warn) != "" {
			warnings = append(warnings, line.Warn)
		}
		distance = lastParsedPositive(distance, line.TMiles, line.Dist, line.LMiles)
	}
	return distance, warnings
}

func lastParsedPositive(current float64, values ...string) float64 {
	for _, value := range values {
		if parsed, ok := parseFloat(value); ok {
			current = parsed
		}
	}
	return current
}

func parseFloat(value string) (float64, bool) {
	var parsed float64
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%f", &parsed); err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func setString(values map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		values[key] = strings.TrimSpace(value)
	}
}

func vehicleTypeCode(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		return 2
	case "lighttruck", "light truck":
		return 1
	default:
		return 0
	}
}

func routingTypeCode(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "shortest":
		return 1
	case "fastest":
		return 2
	default:
		return 0
	}
}

func distanceUnitsCode(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "kilometers", "kilometres", "km":
		return 1
	default:
		return 0
	}
}

func tollRoadsCode(useTolls bool) int {
	if useTolls {
		return 3
	}
	return 2
}

func regionCode(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "AF":
		return 1
	case "AS":
		return 2
	case "EU":
		return 3
	case "OC":
		return 5
	case "SA":
		return 6
	case "ME":
		return 7
	default:
		return 4
	}
}

func hazmatCodes(values []string) []int {
	codes := make([]int, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "general":
			codes = append(codes, 1)
		case "caustic":
			codes = append(codes, 2)
		case "explosives":
			codes = append(codes, 3)
		case "flammable":
			codes = append(codes, 4)
		case "inhalants":
			codes = append(codes, 5)
		case "radioactive":
			codes = append(codes, 6)
		case "harmfultowater", "harmful to water":
			codes = append(codes, 7)
		case "tunnel":
			codes = append(codes, 8)
		}
	}
	return codes
}
