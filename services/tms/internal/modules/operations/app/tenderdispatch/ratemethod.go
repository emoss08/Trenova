package tenderdispatch

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func rateMethodLabel(method shipment.CarrierRateMethod) string {
	return method.Label()
}
