package money

import (
	"strconv"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/shopspring/decimal"
)

func FormatWholeDollars(value decimal.Decimal) string {
	rounded := value.RoundBank(0)
	formatted := "$" + intutils.FormatWithCommas(rounded.Abs().IntPart())
	if rounded.IsNegative() {
		return "-" + formatted
	}
	return formatted
}

func FormatDollars(value decimal.Decimal) string {
	rounded := value.Round(2)
	whole := rounded.Abs().Truncate(0)
	cents := rounded.Abs().Sub(whole).Shift(2).IntPart()
	formatted := "$" + intutils.FormatWithCommas(whole.IntPart()) + "." + twoDigits(cents)
	if rounded.IsNegative() {
		return "-" + formatted
	}
	return formatted
}

func twoDigits(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}
