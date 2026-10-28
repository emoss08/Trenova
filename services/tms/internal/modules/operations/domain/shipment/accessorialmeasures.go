package shipment

import (
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/shopspring/decimal"
)

func (sp *Shipment) AccessorialMeasures() *rateagreement.AccessorialMeasures {
	return &rateagreement.AccessorialMeasures{
		Miles:  sp.RatingMiles(),
		Dwells: sp.DwellHours(),
		Days:   sp.ServiceDays(),
		Stops:  decimal.NewFromInt(int64(sp.RatedStopCount())),
	}
}
