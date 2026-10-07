package decimalutils

import "github.com/shopspring/decimal"

// NullEqual compares two nullable decimals by value, since a decimal
// carries its scale and two spellings of the same money are not == equal.
func NullEqual(a, b decimal.NullDecimal) bool {
	if a.Valid != b.Valid {
		return false
	}

	if !a.Valid {
		return true
	}

	return a.Decimal.Equal(b.Decimal)
}

// PtrEqual compares two optional decimals by value; both absent is equal.
func PtrEqual(a, b *decimal.Decimal) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.Equal(*b)
}
