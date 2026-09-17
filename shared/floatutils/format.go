package floatutils

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/intutils"
)

func FormatGrouped(value float64, maxDecimals int) string {
	text := strconv.FormatFloat(value, 'f', max(maxDecimals, 0), 64)
	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(text, "-")
	whole, fraction, _ := strings.Cut(text, ".")
	fraction = strings.TrimRight(fraction, "0")

	wholeValue, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}

	var sb strings.Builder
	sb.Grow(len(text) + len(whole)/3 + 1)
	if negative && (wholeValue != 0 || fraction != "") {
		sb.WriteByte('-')
	}
	sb.WriteString(intutils.FormatWithCommas(wholeValue))
	if fraction != "" {
		sb.WriteByte('.')
		sb.WriteString(fraction)
	}
	return sb.String()
}

func FormatPercent(value float64, maxDecimals int) string {
	return FormatGrouped(value, maxDecimals) + "%"
}

func FormatFractionAsPercent(fraction float64, maxDecimals int) string {
	return FormatPercent(fraction*100, maxDecimals)
}
