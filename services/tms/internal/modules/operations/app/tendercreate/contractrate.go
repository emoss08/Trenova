package tendercreate

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/shopspring/decimal"
)

// offerPrice is what one guide entry should offer a carrier.
type offerPrice struct {
	method shipment.CarrierRateMethod
	rate   decimal.Decimal
	// reason is why this entry cannot be offered at all. Empty means it can.
	reason string
}
