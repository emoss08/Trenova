package decimalutils

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// ParseMoneyText reads a number the way a spreadsheet writes one.
//
// Currency symbols, thousands separators and trailing percent signs are what a
// person types; refusing them would send somebody back to reformat a file that
// reads perfectly well.
func ParseMoneyText(raw string) (decimal.NullDecimal, error) {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ',', '$', '%', ' ', ' ':
			return -1
		default:
			return r
		}
	}, raw)

	if cleaned == "" {
		return decimal.NullDecimal{}, nil
	}

	value, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.NullDecimal{}, fmt.Errorf("%q is not a number", raw)
	}

	return decimal.NewNullDecimal(value), nil
}
