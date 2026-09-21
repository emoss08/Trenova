package aiprovider

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func money(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)

	return &d
}

// An unknown cost is not a free one. A provider nobody priced yields nil,
// never zero, so a sum cannot understate spend by exactly those providers.
func TestProvider_CostFor(t *testing.T) {
	t.Parallel()

	unpriced := &Provider{}
	assert.False(t, unpriced.Priced())
	assert.Nil(t, unpriced.CostFor(1000, 1000))

	half := &Provider{InputCostPerMillion: money("3")}
	assert.False(t, half.Priced(), "one price is not a price list")
	assert.Nil(t, half.CostFor(1000, 1000))

	claude := &Provider{InputCostPerMillion: money("3"), OutputCostPerMillion: money("15")}
	cost := claude.CostFor(2_000_000, 100_000)
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("7.5")), cost.String())

	free := &Provider{InputCostPerMillion: money("0"), OutputCostPerMillion: money("0")}
	cost = free.CostFor(5000, 5000)
	require.NotNil(t, cost, "a self-hosted model priced at zero is priced")
	assert.True(t, cost.IsZero())
}
