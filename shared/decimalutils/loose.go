package decimalutils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	ErrNumberMissing = errors.New("value is missing")
	ErrNotANumber    = errors.New("is not a number")
)

func ParseLoose(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Zero, ErrNumberMissing
	}

	negative := false
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		negative = true
		value = strings.TrimSuffix(strings.TrimPrefix(value, "("), ")")
	}

	var b strings.Builder
	b.Grow(len(value))
scan:
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r == '.':
			b.WriteRune(r)
		case r == '-':
			negative = !negative
		case r == ',', r == '$', r == ' ', r == ' ', r == '+':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			if b.Len() > 0 {
				break scan
			}
		default:
			return decimal.Zero, fmt.Errorf("%q %w", raw, ErrNotANumber)
		}
	}
	cleaned := b.String()
	if cleaned == "" || cleaned == "." {
		return decimal.Zero, fmt.Errorf("%q %w", raw, ErrNotANumber)
	}

	parsed, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%q %w", raw, ErrNotANumber)
	}
	if negative {
		parsed = parsed.Neg()
	}

	return parsed, nil
}
