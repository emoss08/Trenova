package decimalutils

import "github.com/shopspring/decimal"

// Max returns the larger of two amounts. A decimal carries its scale, so the
// comparison is by value and the returned amount is one of the two inputs
// rather than a recomputed sum.
func Max(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThan(b) {
		return a
	}

	return b
}

// Min returns the smaller of two amounts, by value.
func Min(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	}

	return b
}

// Abs returns the amount without its sign.
func Abs(value decimal.Decimal) decimal.Decimal {
	if value.IsNegative() {
		return value.Neg()
	}

	return value
}
