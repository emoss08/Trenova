package domainvalidation

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetUSD(t *testing.T) {
	t.Parallel()

	rule := BudgetUSD(decimal.NewFromInt(100), "100")

	require.NoError(t, rule.Validate(decimal.NewFromInt(0)))
	require.NoError(t, rule.Validate(decimal.RequireFromString("99.99")))
	require.NoError(t, rule.Validate((*decimal.Decimal)(nil)))

	amount := decimal.NewFromInt(5)
	require.NoError(t, rule.Validate(&amount))

	for value, message := range map[string]string{
		"-0.01":  "A budget cannot be negative",
		"100.01": "A budget is at most 100",
		"1.005":  "A budget is in whole cents",
	} {
		err := rule.Validate(decimal.RequireFromString(value))
		require.Error(t, err, value)
		assert.Equal(t, message, err.Error(), value)
	}

	assert.Error(t, rule.Validate("12"))
}
