package shipment

import (
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
)

func (s *Shipment) RatedByHand() bool {
	return s.RateLocked || s.FormulaTemplateID.IsNil()
}

func (s *Shipment) SetFreightLines(lines []FreightLine) {
	total := decimal.Zero
	for i := range lines {
		total = total.Add(lines[i].Amount)
	}
	s.FreightLines = lines
	s.FreightChargeAmount = decimal.NewNullDecimal(total)
}

func (s *Shipment) ManualFreightLines(label string) []FreightLine {
	amount := decimal.Zero
	if s.FreightChargeAmount.Valid {
		amount = s.FreightChargeAmount.Decimal
	}
	if len(s.FreightLines) == 1 && s.FreightLines[0].Amount.Equal(amount) {
		return s.FreightLines
	}
	return []FreightLine{{Label: label, Amount: amount}}
}

func FreightLinesFromBill(bill *tarifftypes.Bill, adjustmentLabel func(tarifftypes.LineKind) string) []FreightLine {
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
