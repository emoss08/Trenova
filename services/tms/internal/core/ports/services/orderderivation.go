package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// OrderDerivationService recomputes an order's derived state — status (via
// order.Derive over the leg statuses) and the AR total rollup — after one of its legs
// changes. Every code path that mutates a leg's status or charges outside the order
// service must invoke it; the shipment-event observer covers user-driven shipment
// updates, and direct calls cover the paths that do not emit shipment events
// (move lifecycle, invoicing, EDI sync, bulk cancel/delay).
type OrderDerivationService interface {
	// RecomputeOrder re-derives the status and recalculates the total of one order.
	RecomputeOrder(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) error
	// RecomputeForShipment resolves the shipment's parent order and recomputes it.
	// A shipment with no order is a no-op.
	RecomputeForShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) error
}
