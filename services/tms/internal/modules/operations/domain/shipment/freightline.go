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

func (s *Shipment) ManualFreightLines(label string) []FreightLine {
	amount := s.freightAmount()
	if s.RateLocked && len(s.FreightLines) > 0 && sumFreight(s.FreightLines).Equal(amount) {
		return s.FreightLines
	}
	if len(s.FreightLines) == 1 && s.FreightLines[0].Amount.Equal(amount) {
		return s.FreightLines
	}
	return []FreightLine{{Label: label, Amount: amount}}
}

func (s *Shipment) BilledFreightLines(label string) []FreightLine {
	amount := s.freightAmount()
	if len(s.FreightLines) > 0 && sumFreight(s.FreightLines).Equal(amount) {
		return s.FreightLines
	}
	return []FreightLine{{Label: label, Amount: amount}}
}

func (s *Shipment) freightAmount() decimal.Decimal {
	if s.FreightChargeAmount.Valid {
		return s.FreightChargeAmount.Decimal
	}
	return decimal.Zero
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
