package shipmentrating

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/shopspring/decimal"
)

func ManualFreightLines(s *shipment.Shipment, label string) []shipment.FreightLine {
	amount := freightAmount(s)
	if s.RateLocked && len(s.FreightLines) > 0 && freightTotal(s.FreightLines).Equal(amount) {
		return s.FreightLines
	}
	if len(s.FreightLines) == 1 && s.FreightLines[0].Amount.Equal(amount) {
		return s.FreightLines
	}
	return []shipment.FreightLine{{Label: label, Amount: amount}}
}

func BilledFreightLines(s *shipment.Shipment, label string) []shipment.FreightLine {
	amount := freightAmount(s)
	if len(s.FreightLines) > 0 && freightTotal(s.FreightLines).Equal(amount) {
		return s.FreightLines
	}
	return []shipment.FreightLine{{Label: label, Amount: amount}}
}

func freightAmount(s *shipment.Shipment) decimal.Decimal {
	if s.FreightChargeAmount.Valid {
		return s.FreightChargeAmount.Decimal
	}
	return decimal.Zero
}

func freightTotal(lines []shipment.FreightLine) decimal.Decimal {
	total := decimal.Zero
	for i := range lines {
		total = total.Add(lines[i].Amount)
	}
	return total
}
