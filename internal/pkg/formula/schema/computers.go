// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package schema

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

// * RegisterShipmentComputers registers all shipment-related compute functions
func RegisterShipmentComputers(resolver *DefaultDataResolver) {
	resolver.RegisterComputer("computeTemperatureDifferential", computeTemperatureDifferential)
	resolver.RegisterComputer("computeHasHazmat", computeHasHazmat)
	resolver.RegisterComputer(
		"computeRequiresTemperatureControl",
		computeRequiresTemperatureControl,
	)
	resolver.RegisterComputer("computeTotalStops", computeTotalStops)
}

// * computeTemperatureDifferential calculates the difference between max and min temperature
func computeTemperatureDifferential(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return 0.0, nil
	}

	if s.TemperatureMax != nil && s.TemperatureMin != nil {
		return float64(*s.TemperatureMax - *s.TemperatureMin), nil
	}
	return 0.0, nil
}

// * computeHasHazmat checks if any commodity is hazardous
func computeHasHazmat(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, nil
	}

	// * Check if any ShipmentCommodity has a Commodity with HazardousMaterialID
	for _, shipmentCommodity := range s.Commodities {
		if shipmentCommodity.Commodity != nil &&
			shipmentCommodity.Commodity.HazardousMaterialID != nil {
			return true, nil
		}
	}
	return false, nil
}

// * computeRequiresTemperatureControl checks if temperature control is required
func computeRequiresTemperatureControl(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, nil
	}

	return s.TemperatureMin != nil || s.TemperatureMax != nil, nil
}

// * computeTotalStops counts all stops across all moves
func computeTotalStops(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return 0, nil
	}

	totalStops := 0
	for _, move := range s.Moves {
		totalStops += len(move.Stops)
	}
	return totalStops, nil
}
