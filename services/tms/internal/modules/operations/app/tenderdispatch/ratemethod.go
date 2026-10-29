package tenderdispatch

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

func rateMethodLabel(method shipmentvocab.CarrierRateMethod) string {
	return method.Label()
}
