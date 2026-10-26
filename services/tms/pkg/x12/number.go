package x12

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var ErrNotNumber = errors.New("x12: not a number")

func ParseNumber(value, dataType string) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return decimal.Decimal{}, ErrNotNumber
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, ErrNotNumber
	}
	places := impliedPlaces(dataType)
	if places == 0 || strings.Contains(value, ".") {
		return parsed, nil
	}
	return parsed.Shift(-places), nil
}

func impliedPlaces(dataType string) int32 {
	digits, ok := strings.CutPrefix(dataType, "N")
	if !ok || len(digits) != 1 || digits[0] < '0' || digits[0] > '9' {
		return 0
	}
	return int32(digits[0] - '0')
}
