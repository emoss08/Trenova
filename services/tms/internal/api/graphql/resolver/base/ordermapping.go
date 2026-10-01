package base

import (
	"github.com/shopspring/decimal"
)

func NullDecimalToString(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	s := d.Decimal.String()
	return &s
}
