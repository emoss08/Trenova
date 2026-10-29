package carrierpreview

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

func RateMethodWords(method shipmentvocab.CarrierRateMethod) string {
	if method == shipmentvocab.CarrierRateMethodPerMile {
		return "per mile"
	}

	return "flat"
}
