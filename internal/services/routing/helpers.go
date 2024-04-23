// Package routing provides utilities to calculate distances using various methods including API integrations and geometric formulas.
package routing

import (
	"context"
	"log"

	"github.com/emoss08/trenova/internal/ent/routecontrol"
	"github.com/emoss08/trenova/internal/integrations/google"
	"github.com/rs/zerolog"
	"googlemaps.github.io/maps"
)

// DistanceCalculator defines an interface for calculating distances between two points.
type DistanceCalculator interface {
	CalculateDistance(ctx context.Context, point1, point2 Coord, method routecontrol.DistanceMethod, units maps.Units, apiKey string)
}

// DistanceCalculatorImpl implements the DistanceCalculator interface using various methods.
type DistanceCalculatorImpl struct {
	Logger *zerolog.Logger
}

// NewDistanceCalculator creates a new instance of DistanceCalculatorImpl.
func NewDistanceCalculator(logger *zerolog.Logger) *DistanceCalculatorImpl {
	return &DistanceCalculatorImpl{
		Logger: logger,
	}
}

// CalculateDistance calculates the distance between two points using the specified method and units.
func (dc *DistanceCalculatorImpl) CalculateDistance(
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
func (dc *DistanceCalculatorImpl) calculateDistanceMatrix(
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
func (dc *DistanceCalculatorImpl) calculateVincentyDistance(
	p1, p2 Coord,
) {
	miles, kilometers, err := VincentyDistance(p1, p2)
	if err != nil {
		dc.Logger.Error().Err(err).Msg("Failed to calculate distance using Vincenty formula")
		return
	}
	log.Printf("Distance: %.2f miles, %.2f kilometers", miles, kilometers)
}
