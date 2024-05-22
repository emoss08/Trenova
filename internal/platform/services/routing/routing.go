// Package routing provides utilities to calculate distances using various methods including API integrations and geometric formulas.
package routing

import (
	"context"
	"log"

	"github.com/emoss08/trenova/internal/ent/routecontrol"
	"github.com/emoss08/trenova/internal/platform/services/google"
	"github.com/rs/zerolog"
	"googlemaps.github.io/maps"
)

// RoutingService implements the RoutingService interface using various methods.
type RoutingService struct {
	Logger *zerolog.Logger
}

// NewRoutingService creates a new instance of RoutingService.
func NewRoutingService(logger *zerolog.Logger) *RoutingService {
	return &RoutingService{
		Logger: logger,
	}
}

// CalculateDistance calculates the distance between two points using the specified method and units.
func (dc *RoutingService) CalculateDistance(
	ctx context.Context, origins, destinations Coord, method routecontrol.DistanceMethod, units maps.Units, apiKey string,
) {
	switch method {
	case routecontrol.DistanceMethodGoogle:
		originsStr := []string{origins.String()}
		destinationsStr := []string{destinations.String()}

		dc.calculateDistanceMatrix(ctx, originsStr, destinationsStr, units, apiKey)
	case routecontrol.DistanceMethodTrenova:
		dc.calculateVincentyDistance(origins, destinations)
	}
}

// calculateDistanceMatrix calculates the distance and duration using Google Maps API.
func (dc *RoutingService) calculateDistanceMatrix(
	ctx context.Context, origins, destinations []string, units maps.Units, apiKey string,
) {
	resp, err := google.NewClient(dc.Logger).GetDistanceMatrix(ctx, origins, destinations, units, apiKey)
	if err != nil {
		dc.Logger.Error().Err(err).Msg("Failed to calculate distance using Google Maps API")
		return
	}

	for _, row := range resp.Rows {
		for _, element := range row.Elements {
			distance := element.Distance.HumanReadable
			duration := element.Duration.Hours()
			log.Printf("Distance: %s, Duration: %.2f hours", distance, duration)
		}
	}
}

// calculateVincentyDistance calculates the distance using the Vincenty formula.
func (dc *RoutingService) calculateVincentyDistance(
	p1, p2 Coord,
) {
	miles, kilometers, err := VincentyDistance(p1, p2)
	if err != nil {
		dc.Logger.Error().Err(err).Msg("Failed to calculate distance using Vincenty formula")
		return
	}
	log.Printf("Distance: %.2f miles, %.2f kilometers", miles, kilometers)
}

// LocationAutoComplete provides an interface for location auto-completion
func (dc *RoutingService) LocationAutoComplete(ctx context.Context, query, apiKey string) ([]map[string]any, error) {
	client, err := google.NewClient(dc.Logger).GetClientForOrganization(apiKey)
	if err != nil {
		dc.Logger.Printf("Error creating Google Maps client: %v", err)
		return nil, err
	}

	// Default values for the location auto-complete request
	req := &maps.QueryAutocompleteRequest{
		Input:    query,
		Language: "en",
	}

	resp, err := client.QueryAutocomplete(ctx, req)
	if err != nil {
		dc.Logger.Printf("Error with QueryAutocomplete request: %v", err)
		return nil, err
	}

	// Extract the location suggestions and fetch detailed information
	detailedResults := make([]map[string]any, 0)
	for _, prediction := range resp.Predictions {
		detailReq := &maps.PlaceDetailsRequest{
			PlaceID: prediction.PlaceID,
		}
		details, err := client.PlaceDetails(ctx, detailReq)
		if err != nil {
			dc.Logger.Printf("Error fetching place details for %s: %v", prediction.PlaceID, err)
			continue // Log error and skip this prediction
		}

		addressComponents := make(map[string]string)
		for _, component := range details.AddressComponents {
			addressComponents[component.Types[0]] = component.LongName
		}

		placeDetails := map[string]any{
			"name":               prediction.StructuredFormatting.MainText,
			"address":            details.FormattedAddress,
			"address_components": addressComponents,
			"place_id":           prediction.PlaceID,
		}
		detailedResults = append(detailedResults, placeDetails)
	}

	return detailedResults, nil
}
