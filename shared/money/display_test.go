package money

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDisplayWritesAnAmountAsAnInvoiceShowsIt(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		currency string
		value    string
		want     string
	}{
		{"USD", "10265.22", "$10,265.22"},
		{"usd", "917", "$917.00"},
		{"", "0.5", "$0.50"},
		{"USD", "-1234.567", "-$1,234.57"},
		{"CAD", "1500", "CA$1,500.00"},
		{"EUR", "99.999", "€100.00"},
		{"JPY", "1000000", "JPY 1,000,000.00"},
		{"USD", "-0.001", "$0.00"},
	} {
		assert.Equal(t, tc.want, Display(tc.currency, decimal.RequireFromString(tc.value)),
			"%s %s", tc.currency, tc.value)
	}
}
