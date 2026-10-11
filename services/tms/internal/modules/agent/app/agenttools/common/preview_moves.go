package common

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
)

func FindStop(move *shipment.ShipmentMove, stopID pulid.ID) *shipment.Stop {
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID {
			return stop
		}
	}

	return nil
}

func StopLabel(stop *shipment.Stop) string {
	if stop == nil {
		return "Stop"
	}
	if stop.Location != nil && stop.Location.Name != "" {
		return fmt.Sprintf("%s at %s", stop.Type, stop.Location.Name)
	}

	return fmt.Sprintf("%s stop %d", stop.Type, stop.Sequence+1)
}
