// Package settlementshared holds the accrual, period, posting-workflow, and
// CSV internals shared byte-for-byte between the driver and carrier
// settlement services. Behavior changes here affect both payment rails.
package settlementshared

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

//nolint:exhaustive // canceled shipments never accrue pay or cost
var shipmentStatusRank = map[shipment.Status]int{
	shipment.StatusNew:                0,
	shipment.StatusPartiallyAssigned:  1,
	shipment.StatusAssigned:           2,
	shipment.StatusInTransit:          3,
	shipment.StatusDelayed:            3,
	shipment.StatusPartiallyCompleted: 4,
	shipment.StatusCompleted:          5,
	shipment.StatusReadyToInvoice:     6,
	shipment.StatusInvoiced:           7,
}

// ShipmentStatusRank returns the monotonic progression rank for a shipment
// status; ok is false for statuses that never accrue (e.g. canceled).
func ShipmentStatusRank(status shipment.Status) (rank int, ok bool) {
	rank, ok = shipmentStatusRank[status]
	return rank, ok
}

// PayTriggerRank maps a settlement pay trigger to the shipment-status rank at
// which accrual fires.
func PayTriggerRank(trigger settlementcontrol.PayTrigger) int {
	switch trigger {
	case settlementcontrol.PayTriggerMoveCompleted, settlementcontrol.PayTriggerShipmentDelivered:
		return shipmentStatusRank[shipment.StatusCompleted]
	case settlementcontrol.PayTriggerPODReceived:
		return shipmentStatusRank[shipment.StatusReadyToInvoice]
	case settlementcontrol.PayTriggerShipmentInvoiced:
		return shipmentStatusRank[shipment.StatusInvoiced]
	default:
		return shipmentStatusRank[shipment.StatusCompleted]
	}
}
