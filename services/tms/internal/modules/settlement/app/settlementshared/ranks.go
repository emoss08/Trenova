// Package settlementshared holds the accrual, period, posting-workflow, and
// CSV internals shared byte-for-byte between the driver and carrier
// settlement services. Behavior changes here affect both payment rails.
package settlementshared

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

//nolint:exhaustive // canceled shipments never accrue pay or cost
var shipmentStatusRank = map[shipmentvocab.Status]int{
	shipmentvocab.StatusNew:                0,
	shipmentvocab.StatusPartiallyAssigned:  1,
	shipmentvocab.StatusAssigned:           2,
	shipmentvocab.StatusInTransit:          3,
	shipmentvocab.StatusDelayed:            3,
	shipmentvocab.StatusPartiallyCompleted: 4,
	shipmentvocab.StatusCompleted:          5,
	shipmentvocab.StatusReadyToInvoice:     6,
	shipmentvocab.StatusInvoiced:           7,
}

// ShipmentStatusRank returns the monotonic progression rank for a shipment
// status; ok is false for statuses that never accrue (e.g. canceled).
func ShipmentStatusRank(status shipmentvocab.Status) (rank int, ok bool) {
	rank, ok = shipmentStatusRank[status]
	return rank, ok
}

// PayTriggerRank maps a settlement pay trigger to the shipment-status rank at
// which accrual fires.
func PayTriggerRank(trigger settlementcontrol.PayTrigger) int {
	switch trigger {
	case settlementcontrol.PayTriggerMoveCompleted, settlementcontrol.PayTriggerShipmentDelivered:
		return shipmentStatusRank[shipmentvocab.StatusCompleted]
	case settlementcontrol.PayTriggerPODReceived:
		return shipmentStatusRank[shipmentvocab.StatusReadyToInvoice]
	case settlementcontrol.PayTriggerShipmentInvoiced:
		return shipmentStatusRank[shipmentvocab.StatusInvoiced]
	default:
		return shipmentStatusRank[shipmentvocab.StatusCompleted]
	}
}
