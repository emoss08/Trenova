package shipmentrating

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
)

func SplitFreightShare(lines []shipment.FreightLine, share decimal.Decimal) []decimal.Decimal {
	out := make([]decimal.Decimal, len(lines))
	if len(lines) == 0 {
		return out
	}
	total := decimal.Zero
	for i := range lines {
		total = total.Add(lines[i].Amount)
	}
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
		out[i] = lines[i].Amount.Mul(share).Div(total).Round(chargeallocation.SharePlaces)
		remaining = remaining.Sub(out[i])
	}
	out[len(lines)-1] = remaining
	return out
}

func FreightLinesFromBill(
	bill *tarifftypes.Bill,
	adjustmentLabel func(tarifftypes.LineKind) string,
) []shipment.FreightLine {
	if bill == nil {
		return []shipment.FreightLine{}
	}
	lines := make([]shipment.FreightLine, 0, len(bill.Lines))
	for i := range bill.Lines {
		line := &bill.Lines[i]
		if line.Code != "" {
			continue
		}
		label := line.Label
		if label == "" && adjustmentLabel != nil {
			label = adjustmentLabel(line.Kind)
		}
		lines = append(lines, shipment.FreightLine{
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
