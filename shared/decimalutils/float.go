package decimalutils

import (
	"math"
	"strconv"

	"github.com/shopspring/decimal"
)

const significantDigits = 15

func FromFloat(value float64) (decimal.Decimal, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return decimal.Zero, false
	}
	parsed, err := decimal.NewFromString(strconv.FormatFloat(value, 'g', significantDigits, 64))
	if err != nil {
		return decimal.Zero, false
	}
	return parsed, true
}
