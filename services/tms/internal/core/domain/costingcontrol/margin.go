package costingcontrol

import "github.com/shopspring/decimal"

var DefaultTargetMarginPercent = decimal.NewFromInt(10)

func EffectiveTargetMarginPercent(target decimal.NullDecimal) decimal.Decimal {
	if !target.Valid || !target.Decimal.IsPositive() {
		return DefaultTargetMarginPercent
	}
	return target.Decimal
}
