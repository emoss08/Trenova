package schema

import "github.com/emoss08/trenova/internal/core/domain/shipment"

func RegisterShipmentComputers(resolver *DefaultDataResolver) {
	resolver.RegisterComputer("computeTemperatureDifferential", computeTemperatureDifferential)
	resolver.RegisterComputer("computeHasHazmat", computeHasHazmat)
	resolver.RegisterComputer(
		"computeRequiresTemperatureControl",
		computeRequiresTemperatureControl,
	)
	resolver.RegisterComputer("computeTotalStops", computeTotalStops)
}

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

func computeHasHazmat(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, nil
	}

	for _, shipmentCommodity := range s.Commodities {
		if shipmentCommodity.Commodity != nil &&
			shipmentCommodity.Commodity.HazardousMaterialID != nil {
			return true, nil
		}
	}
	return false, nil
}

func computeRequiresTemperatureControl(entity any) (any, error) {
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, nil
	}

	return s.TemperatureMin != nil || s.TemperatureMax != nil, nil
}

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
