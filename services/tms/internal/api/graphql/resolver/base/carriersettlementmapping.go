package base

import (
	"github.com/shopspring/decimal"
)

func NullDecimalStringPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Decimal.String()
	return &formatted
}
