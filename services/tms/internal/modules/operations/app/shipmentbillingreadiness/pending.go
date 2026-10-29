package shipmentbillingreadiness

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

// PendingReadiness is a shipment that passed the gate and waits on its
// readiness.
type PendingReadiness struct {
	Index      int
	Entity     *shipment.Shipment
	Resolution *chargeallocation.ShareResolution
	MarkReady  bool
}
