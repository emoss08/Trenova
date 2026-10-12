package common

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func RateMethodWords(method shipment.CarrierRateMethod) string {
	if method == shipment.CarrierRateMethodPerMile {
		return "per mile"
	}

	return "flat"
}
