package rateagreement_test

import "github.com/shopspring/decimal"

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func nullDec(value string) decimal.NullDecimal {
	if value == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(dec(value))
}
