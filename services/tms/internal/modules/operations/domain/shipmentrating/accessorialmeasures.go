package shipmentrating

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/shopspring/decimal"
)

func AccessorialMeasures(sp *shipment.Shipment) *rateagreement.AccessorialMeasures {
	return &rateagreement.AccessorialMeasures{
		Miles:  Miles(sp),
		Dwells: DwellHours(sp),
		Days:   ServiceDays(sp),
		Stops:  decimal.NewFromInt(int64(RatedStopCount(sp))),
	}
}
