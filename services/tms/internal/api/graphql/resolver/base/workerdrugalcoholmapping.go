package base

import (
	"github.com/shopspring/decimal"
)

func DecimalPtrString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	rendered := value.String()
	return &rendered
}

// nullDecimalValue unwraps an optional decimal into the pointer the domain
// models use for a column that may be absent. Its sibling nullDecimalPtr
// renders one for the wire, which is the opposite direction.
func NullDecimalValue(value decimal.NullDecimal) *decimal.Decimal {
	if !value.Valid {
		return nil
	}
	amount := value.Decimal
	return &amount
}
