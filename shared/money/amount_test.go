package money

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestMinorUnitsUsesBankersRounding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		value    string
		expected int64
	}{
		{name: "whole dollars", value: "123.45", expected: 12345},
		{name: "half to even rounds down", value: "1.005", expected: 100},
		{name: "half to even rounds up", value: "1.015", expected: 102},
		{name: "negative half to even", value: "-1.015", expected: -102},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, MinorUnits(decimal.RequireFromString(tc.value)))
		})
	}
}

func TestFromDecimalDefaultsCurrencyCode(t *testing.T) {
	t.Parallel()

	amount := FromDecimal("", decimal.RequireFromString("10.25"))

	assert.Equal(t, DefaultCurrencyCode, amount.CurrencyCode)
	assert.Equal(t, int64(1025), amount.Minor)
}

func TestDecimalFromMinor(t *testing.T) {
	t.Parallel()

	assert.True(t, decimal.RequireFromString("10.25").Equal(DecimalFromMinor(1025)))
	assert.True(t, decimal.RequireFromString("-3.50").Equal(DecimalFromMinor(-350)))
}
