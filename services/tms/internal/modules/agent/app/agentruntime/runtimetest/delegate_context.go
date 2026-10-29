package runtimetest

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/shared/pulid"
)

func ShipmentDesk() agentturn.RuntimeDelegate {
	return agentturn.RuntimeDelegate{
		ID:          pulid.MustNew("agdef_"),
		Name:        "Shipment Desk",
		Description: "Creates and changes shipments.",
		Tools:       []string{"shipment_duplicate", "shipment_get"},
	}
}
