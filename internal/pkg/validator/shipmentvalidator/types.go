package shipmentvalidator

import "github.com/emoss08/trenova/internal/core/domain/shipment"

var cancelableShipmentStatuses = map[shipment.Status]bool{
	shipment.StatusNew:       true,  // Can cancel new shipments
	shipment.StatusInTransit: true,  // Can cancel in-transit shipments
	shipment.StatusDelayed:   true,  // Can cancel delayed shipments
	shipment.StatusCompleted: false, // Can't cancel completed shipments
	shipment.StatusBilled:    false, // Can't cancel billed shipments
	shipment.StatusCanceled:  false, // Can't cancel already canceled shipments
}
