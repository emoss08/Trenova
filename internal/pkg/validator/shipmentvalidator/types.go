package shipmentvalidator

import "github.com/emoss08/trenova/internal/core/domain/shipment"

var cancelableShipmentStatuses = map[shipment.Status]bool{
	shipment.StatusNew:                true,  // Can cancel new shipments
	shipment.StatusPartiallyAssigned:  true,  // Can cancel partially assigned shipments
	shipment.StatusAssigned:           true,  // Can cancel assigned shipments
	shipment.StatusInTransit:          true,  // Can cancel in-transit shipments
	shipment.StatusDelayed:            true,  // Can cancel delayed shipments
	shipment.StatusPartiallyCompleted: true,  // Can cancel partially completed shipments
	shipment.StatusCompleted:          true,  // Can cancel completed shipments
	shipment.StatusBilled:             false, // Can't cancel billed shipments
	shipment.StatusCanceled:           false, // Can't cancel already canceled shipments
}
