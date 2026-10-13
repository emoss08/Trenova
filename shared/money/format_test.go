package money_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFormatWholeDollars(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"0":          "$0",
		"750":        "$750",
		"75000":      "$75,000",
		"5000000":    "$5,000,000",
		"1234567.50": "$1,234,568",
		"-500":       "-$500",
		"-12500.4":   "-$12,500",
	}
	for input, want := range cases {
		assert.Equal(t, want, money.FormatWholeDollars(decimal.RequireFromString(input)), input)
	}
}

func TestFormatDollars(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"0":          "$0.00",
		"5.5":        "$5.50",
		"1234.567":   "$1,234.57",
		"1000000.05": "$1,000,000.05",
		"-1250.4":    "-$1,250.40",
		"-0.001":     "$0.00",
	}
	for input, want := range cases {
		assert.Equal(t, want, money.FormatDollars(decimal.RequireFromString(input)), input)
	}
}
