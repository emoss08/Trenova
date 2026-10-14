package i18n

import (
	"math"
	"strconv"
	"strings"
)

type numberSymbols struct {
	group   string
	decimal string
}

var localeNumberSymbols = map[Locale]numberSymbols{
	EN:   {group: ",", decimal: "."},
	ES:   {group: ",", decimal: "."},
	ZhTW: {group: ",", decimal: "."},
	ZhCN: {group: ",", decimal: "."},
}

const maxCountFractionDigits = 3

func formatCount(locale Locale, value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	symbols, ok := localeNumberSymbols[locale]
	if !ok {
		symbols = localeNumberSymbols[Default]
	}

	text := strconv.FormatFloat(math.Abs(value), 'f', maxCountFractionDigits, 64)
	whole, fraction, _ := strings.Cut(text, ".")
	fraction = strings.TrimRight(fraction, "0")

	var out strings.Builder
	out.Grow(len(text) + len(whole)/3 + 1)
	if value < 0 && (whole != "0" || fraction != "") {
		out.WriteByte('-')
	}
	for idx, digit := range whole {
		if idx > 0 && (len(whole)-idx)%3 == 0 {
			out.WriteString(symbols.group)
		}
		out.WriteRune(digit)
	}
	if fraction != "" {
		out.WriteString(symbols.decimal)
		out.WriteString(fraction)
	}

	return out.String()
}
