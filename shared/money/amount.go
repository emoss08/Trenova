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

func DecimalFromMinor(minor int64) decimal.Decimal {
	return decimal.NewFromInt(minor).Shift(-2)
}

func FormatMinor(minor int64, currencyCode string) string {
	return DecimalFromMinor(minor).StringFixed(2) + " " + normalizeCurrencyCode(currencyCode)
}

// FormatDecimal renders an amount as "USD 1234.50".
//
// The currency leads because these strings go onto customer-facing documents
// where a bare number invites a dispute about which currency was billed.
func FormatDecimal(currencyCode string, value decimal.Decimal) string {
	return normalizeCurrencyCode(currencyCode) + " " + value.StringFixed(2)
}

// FormatDecimalString renders an already-formatted amount with its currency.
// It exists for callers holding the string form of a decimal, so they do not
// re-parse it only to format it again.
func FormatDecimalString(currencyCode, amount string) string {
	return normalizeCurrencyCode(currencyCode) + " " + amount
}

func normalizeCurrencyCode(currencyCode string) string {
	code := strings.ToUpper(strings.TrimSpace(currencyCode))
	if code == "" {
		return DefaultCurrencyCode
	}

	return code
}
