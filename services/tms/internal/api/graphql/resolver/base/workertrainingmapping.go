package base

import (
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func NullDecimalPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	return stringutils.Ptr(value.Decimal.StringFixed(2))
}
