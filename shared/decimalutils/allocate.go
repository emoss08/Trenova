package decimalutils

import (
	"errors"

	"github.com/shopspring/decimal"
)

const percentComparisonPlaces = 6

var (
	Percent100 = decimal.NewFromInt(100)

	ErrNoPercents         = errors.New("at least one percent is required")
	ErrPercentOutOfRange  = errors.New("each percent must be greater than 0 and at most 100")
	ErrPercentsDoNotTotal = errors.New("percents must total 100")
)

// AllocatePercent splits total across percents, rounding every share to places
// with bankers rounding and folding any rounding remainder into the last share
// so the parts always sum back to total exactly.
func AllocatePercent(
	total decimal.Decimal,
	percents []decimal.Decimal,
	places int32,
) ([]decimal.Decimal, error) {
	if len(percents) == 0 {
		return nil, ErrNoPercents
	}

	sum := decimal.Zero
	for _, percent := range percents {
		if percent.LessThanOrEqual(decimal.Zero) || percent.GreaterThan(Percent100) {
			return nil, ErrPercentOutOfRange
		}
		sum = sum.Add(percent)
	}
	if !sum.Round(percentComparisonPlaces).Equal(Percent100) {
		return nil, ErrPercentsDoNotTotal
	}

	shares := make([]decimal.Decimal, len(percents))
	allocated := decimal.Zero
	last := len(percents) - 1
	for i := 0; i < last; i++ {
		shares[i] = total.Mul(percents[i]).Div(Percent100).RoundBank(places)
		allocated = allocated.Add(shares[i])
	}
	shares[last] = total.Sub(allocated)

	return shares, nil
}

// SumEquals reports whether parts add up to total when both sides are compared
// at places precision.
func SumEquals(parts []decimal.Decimal, total decimal.Decimal, places int32) bool {
	sum := decimal.Zero
	for _, part := range parts {
		sum = sum.Add(part)
	}

	return sum.RoundBank(places).Equal(total.RoundBank(places))
}
