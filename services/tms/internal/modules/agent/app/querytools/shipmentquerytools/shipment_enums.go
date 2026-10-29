package shipmentquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	listShipmentStatuses = agenttoolschema.Source("shipment.status", shipmentStatuses)
	DetailLevels         = agenttoolschema.Source("shipment.detail", []Detail{
		DetailSummary,
		DetailFull,
	})
)
