package invoicedocument

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func ShipmentRoute(shp *shipment.Shipment) string {
	origin := ShipmentOrigin(shp)
	destination := ShipmentDestination(shp)
	if origin == "" && destination == "" {
		return ""
	}
	return origin + " -> " + destination
}

func ShipmentOrigin(shp *shipment.Shipment) string {
	return stopLocationName(firstPickupStop(shp))
}

func ShipmentDestination(shp *shipment.Shipment) string {
	return stopLocationName(finalDeliveryStop(shp))
}

func firstPickupStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsOriginStop() },
		preferLowerStopSequence,
	)
}

func firstDeliveryStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsDestinationStop() },
		preferLowerStopSequence,
	)
}

func finalDeliveryStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsDestinationStop() },
		preferHigherStopSequence,
	)
}

func selectShipmentStop(
	shp *shipment.Shipment,
	matches func(*shipment.Stop) bool,
	prefer func(candidate, selected *shipment.Stop) bool,
) *shipment.Stop {
	if shp == nil {
		return nil
	}
	var selected *shipment.Stop
	for _, move := range shp.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if !matches(stop) {
				continue
			}
			if prefer(stop, selected) {
				selected = stop
			}
		}
	}
	return selected
}

func preferLowerStopSequence(candidate, selected *shipment.Stop) bool {
	return selected == nil || candidate.Sequence < selected.Sequence
}

func preferHigherStopSequence(candidate, selected *shipment.Stop) bool {
	return selected == nil || candidate.Sequence > selected.Sequence
}

func stopLocationName(stop *shipment.Stop) string {
	if stop == nil {
		return ""
	}
	if stop.Location != nil {
		return strings.TrimSpace(strings.Join([]string{
			stop.Location.Name,
			stop.Location.City,
			stop.Location.PostalCode,
		}, " "))
	}
	return strings.TrimSpace(stop.AddressLine)
}

func CommoditySummary(shp *shipment.Shipment) string {
	if shp == nil || len(shp.Commodities) == 0 {
		return ""
	}
	parts := make([]string, 0, len(shp.Commodities))
	for _, item := range shp.Commodities {
		if item == nil {
			continue
		}
		name := "Commodity"
		if item.Commodity != nil && strings.TrimSpace(item.Commodity.Name) != "" {
			name = item.Commodity.Name
		}
		parts = append(parts, fmt.Sprintf("%s (%d pcs, %d lbs)", name, item.Pieces, item.Weight))
	}
	return strings.Join(parts, "; ")
}

func ShipmentPro(shp *shipment.Shipment) string {
	if shp == nil {
		return ""
	}
	return shp.ProNumber
}

func ShipmentBOL(shp *shipment.Shipment) string {
	if shp == nil {
		return ""
	}
	return shp.BOL
}

func Int64PtrString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
