package shipment

import (
	"github.com/emoss08/trenova/pkg/tarifftypes"
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

func SplitFreightShare(lines []FreightLine, share decimal.Decimal) []decimal.Decimal {
	out := make([]decimal.Decimal, len(lines))
	if len(lines) == 0 {
		return out
	}
	total := sumFreight(lines)
	if share.Equal(total) {
		for i := range lines {
			out[i] = lines[i].Amount
		}
		return out
	}
	remaining := share
	for i := range lines[:len(lines)-1] {
		if total.IsZero() {
			out[i] = decimal.Zero
			continue
		}
		out[i] = lines[i].Amount.Mul(share).Div(total).Round(SharePlaces)
		remaining = remaining.Sub(out[i])
	}
	out[len(lines)-1] = remaining
	return out
}

func FreightLinesFromBill(
	bill *tarifftypes.Bill,
	adjustmentLabel func(tarifftypes.LineKind) string,
) []FreightLine {
	if bill == nil {
		return []FreightLine{}
	}
	lines := make([]FreightLine, 0, len(bill.Lines))
	for i := range bill.Lines {
		line := &bill.Lines[i]
		if line.Code != "" {
			continue
		}
		label := line.Label
		if label == "" && adjustmentLabel != nil {
			label = adjustmentLabel(line.Kind)
		}
		lines = append(lines, FreightLine{
			Label:        label,
			Quantity:     line.Quantity,
			QuantityUnit: line.QuantityUnit,
			Rate:         line.Rate,
			RateUnit:     line.RateUnit,
			Amount:       line.Amount,
		})
	}
	return lines
}

const RatingSourceManual = "ManualOverride"

func (d *RatingDetail) RatedByHand() bool {
	return d != nil && d.Source == RatingSourceManual
}
