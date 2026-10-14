package formula

import (
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
)

const (
	RoundingRemainderLine  = "roundingRemainder"
	roundingRemainderLabel = "Rounding"
)

var BreakdownReconcileTolerance = decimal.New(5, -3)

func RoundBreakdown(
	policy formulatypes.ChargePolicy,
	lines []formulatemplatetypes.BreakdownAmount,
	rawAmount decimal.Decimal,
) []formulatemplatetypes.BreakdownAmount {
	if len(lines) == 0 {
		return lines
	}

	policy = policy.Normalized()
	rounded := make([]formulatemplatetypes.BreakdownAmount, 0, len(lines)+1)
	unroundedTotal := decimal.Zero
	roundedTotal := decimal.Zero
	failed := false

	for _, line := range lines {
		if line.Error != "" {
			failed = true
			rounded = append(rounded, line)
			continue
		}
		unroundedTotal = unroundedTotal.Add(line.Amount)
		line.Amount = policy.RoundingMode.Round(line.Amount, policy.RoundingPrecision)
		roundedTotal = roundedTotal.Add(line.Amount)
		rounded = append(rounded, line)
	}

	if failed || unroundedTotal.Sub(rawAmount).Abs().GreaterThan(BreakdownReconcileTolerance) {
		return rounded
	}

	remainder := policy.RoundingMode.
		Round(rawAmount, policy.RoundingPrecision).
		Sub(roundedTotal)
	if remainder.IsZero() {
		return rounded
	}

	return append(rounded, formulatemplatetypes.BreakdownAmount{
		Name:   RoundingRemainderLine,
		Label:  roundingRemainderLabel,
		Amount: remainder,
	})
}
