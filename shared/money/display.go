package money

import (
	"strings"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/shopspring/decimal"
)

var currencySymbols = map[string]string{
	"USD": "$",
	"CAD": "CA$",
	"MXN": "MX$",
	"EUR": "€",
	"GBP": "£",
}

func Display(currencyCode string, value decimal.Decimal) string {
	code := normalizeCurrencyCode(currencyCode)
	rounded := value.Round(2)
	whole := rounded.Abs().Truncate(0)
	cents := rounded.Abs().Sub(whole).Shift(2).IntPart()
	figure := intutils.FormatWithCommas(whole.IntPart()) + "." + twoDigits(cents)

	var b strings.Builder
	if rounded.IsNegative() {
		b.WriteByte('-')
	}
	if symbol, ok := currencySymbols[code]; ok {
		b.WriteString(symbol)
		b.WriteString(figure)
		return b.String()
	}
	b.WriteString(code)
	b.WriteByte(' ')
	b.WriteString(figure)
	return b.String()
}
