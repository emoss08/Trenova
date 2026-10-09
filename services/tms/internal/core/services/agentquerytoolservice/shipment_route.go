package agentquerytoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

const coverageNone = "needs a driver"

type shipmentRoute struct {
	pickup   string
	delivery string
	coverage string
}

func routeOf(item *shipment.Shipment) shipmentRoute {
	route := shipmentRoute{}
	if item == nil || item.Moves == nil {
		return route
	}

	var first, last *shipment.Stop
	crews := make([]string, 0, len(item.Moves))
	moves, open := 0, 0
	for _, move := range item.Moves {
		if move == nil || move.Status == shipment.MoveStatusCanceled {
			continue
		}
		moves++
		if crew := moveCrew(move); crew != "" {
			crews = append(crews, crew)
		} else {
			open++
		}
		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipment.StopStatusCanceled {
				continue
			}
			if first == nil && isPickup(stop.Type) {
				first = stop
			}
			if isDelivery(stop.Type) {
				last = stop
			}
		}
	}

	if first != nil {
		route.pickup = stopLine(first)
	}
	if last != nil {
		route.delivery = stopLine(last)
	}
	route.coverage = coverageLine(crews, moves, open)

	return route
}

func moveCrew(move *shipment.ShipmentMove) string {
	if assignment := move.Assignment; assignment != nil && assignment.PrimaryWorkerID != nil &&
		assignment.PrimaryWorkerID.IsNotNil() {
		name := "a driver"
		if worker := assignment.PrimaryWorker; worker != nil {
			name = strings.TrimSpace(worker.FirstName + " " + worker.LastName)
		}
		if tractor := assignment.Tractor; tractor != nil && tractor.Code != "" {
			name += " on " + tractor.Code
		}

		return name
	}
	if move.CarrierAssignment != nil {
		return "a carrier"
	}

	return ""
}

func isPickup(stopType shipment.StopType) bool {
	return stopType == shipment.StopTypePickup || stopType == shipment.StopTypeSplitPickup
}

func isDelivery(stopType shipment.StopType) bool {
	return stopType == shipment.StopTypeDelivery || stopType == shipment.StopTypeSplitDelivery
}

func stopLine(stop *shipment.Stop) string {
	summary := stopSummary(stop, "")

	place := summary.Location
	if summary.City != "" {
		town := summary.City
		if summary.State != "" {
			town += " " + summary.State
		}
		if place == "" {
			place = town
		} else {
			place += ", " + town
		}
	}

	parts := make([]string, 0, 3)
	if place != "" {
		parts = append(parts, place)
	}
	if summary.WindowStart != "" {
		window := summary.WindowStart
		if summary.WindowEnd != "" {
			window += " to " + summary.WindowEnd
		}
		parts = append(parts, window+" "+summary.Timezone)
	}
	if summary.ActualArrival != "" {
		parts = append(parts, "arrived "+summary.ActualArrival)
	}

	return strings.Join(parts, " · ")
}

func coverageLine(crews []string, moves, open int) string {
	switch {
	case moves == 0:
		return ""
	case open == 0:
		return strings.Join(crews, "; ")
	case open == moves:
		return coverageNone
	default:
		return fmt.Sprintf("%s; needs a driver on %d of %d moves", strings.Join(crews, "; "), open, moves)
	}
}
