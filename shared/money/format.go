package money

import (
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
