package inboundqueue

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
)

// ShipmentFinder and PartyFinder are the lookups matching and linking need.
// They are the repository ports under the names this package reads them by.
type (
	ShipmentFinder = inboundmessageports.InboundShipmentFinder
	PartyFinder    = inboundmessageports.InboundPartyFinder
)
