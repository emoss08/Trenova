package shipment

import (
	"github.com/shopspring/decimal"
)

func (s *Shipment) RatedByHand() bool {
	return s.RateLocked || s.FormulaTemplateID.IsNil()
}

func (s *Shipment) SetFreightLines(lines []FreightLine) {
	s.FreightLines = lines
	s.FreightChargeAmount = decimal.NewNullDecimal(sumFreight(lines))
}

func sumFreight(lines []FreightLine) decimal.Decimal {
	total := decimal.Zero
	for i := range lines {
		total = total.Add(lines[i].Amount)
	}
	return total
}

const RatingSourceManual = "ManualOverride"

func (d *RatingDetail) RatedByHand() bool {
	return d != nil && d.Source == RatingSourceManual
}
