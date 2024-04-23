package routing

import (
	"context"
	"log"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent/routecontrol"
	"github.com/emoss08/trenova/internal/integrations/google"
	"googlemaps.github.io/maps"
)

type DistanceCalculator interface {
	CalculateDistance(point1, point2 string, distanceMethod routecontrol.DistanceMethod)
}

type DistanceCalculatorImpl struct {
	Server *api.Server
}

func NewDistanceCalculator(s *api.Server) *DistanceCalculatorImpl {
	return &DistanceCalculatorImpl{
		Server: s,
	}
}

func (dc *DistanceCalculatorImpl) CalculateDistance(
	ctx context.Context, origins, destinations []string, distanceMethod routecontrol.DistanceMethod, units maps.Units, apiKey string,
) {
	if distanceMethod == "Google" {
		// Calculate distance using Google Maps API

		resp, err := google.NewClient(dc.Server).GetDistanceMatrix(ctx, origins, destinations, units, apiKey)

		var distance string
		var duration float64

		if err != nil {
			dc.Server.Logger.Error().Err(err).Msg("Error calculating distance using Google Maps API")
			return
		}

		for _, row := range resp.Rows {
			for _, element := range row.Elements {
				distance = element.Distance.HumanReadable
				duration = element.Duration.Hours()
			}
		}

		log.Printf("Distance: %s, Duration: %f hours", distance, duration)

	} else {
		log.Println("Calculating distance using Trenova API")
		// Calculate distance using Trenova API
	}
}
