package money

import (
	"strings"

	"github.com/shopspring/decimal"
)

const DefaultCurrencyCode = "USD"

type Amount struct {
	CurrencyCode string
	Minor        int64
}

func New(currencyCode string, minor int64) Amount {
	return Amount{
		CurrencyCode: normalizeCurrencyCode(currencyCode),
		Minor:        minor,
	}
}

func FromDecimal(currencyCode string, value decimal.Decimal) Amount {
	return Amount{
		CurrencyCode: normalizeCurrencyCode(currencyCode),
		Minor:        MinorUnits(value),
	}
}

func MinorUnits(value decimal.Decimal) int64 {
	return value.RoundBank(2).Shift(2).IntPart()
}

func normalizeCurrencyCode(currencyCode string) string {
	code := strings.ToUpper(strings.TrimSpace(currencyCode))
	if code == "" {
		return DefaultCurrencyCode
	}

	return code
}
