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
	assert.Nil(t, unpriced.CostFor(TokenUsage{Input: 1000, Output: 1000}))

	half := &Provider{InputCostPerMillion: money("3")}
	assert.False(t, half.Priced(), "one price is not a price list")
	assert.Nil(t, half.CostFor(TokenUsage{Input: 1000, Output: 1000}))

	claude := &Provider{InputCostPerMillion: money("3"), OutputCostPerMillion: money("15")}
	cost := claude.CostFor(TokenUsage{Input: 2_000_000, Output: 100_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("7.5")), cost.String())

	free := &Provider{InputCostPerMillion: money("0"), OutputCostPerMillion: money("0")}
	cost = free.CostFor(TokenUsage{Input: 5000, Output: 5000})
	require.NotNil(t, cost, "a self-hosted model priced at zero is priced")
	assert.True(t, cost.IsZero())
}

/*
A Haiku Desk turn read "4 in" tokens: Anthropic counts what it served from the
cache, and wrote to it, beside input_tokens, so the cached prompt was never
charged. Read at a tenth of input and written at one and a quarter times it,
unless a price is entered.
*/
func TestProvider_CostFor_PricesAnthropicCacheBesideInput(t *testing.T) {
	t.Parallel()

	haiku := &Provider{
		Kind:                 KindAnthropicMessages,
		InputCostPerMillion:  money("1"),
		OutputCostPerMillion: money("5"),
	}
	cost := haiku.CostFor(TokenUsage{
		Input: 4, Output: 200, CacheRead: 1_000_000, CacheWrite: 100_000,
	})
	require.NotNil(t, cost)
	want := decimal.RequireFromString("0.000004").
		Add(decimal.RequireFromString("0.001")).
		Add(decimal.RequireFromString("0.1")).
		Add(decimal.RequireFromString("0.125"))
	assert.True(t, cost.Equal(want), cost.String())

	haiku.CacheReadCostPerMillion = money("0.08")
	haiku.CacheWriteCostPerMillion = money("2")
	cost = haiku.CostFor(TokenUsage{CacheRead: 1_000_000, CacheWrite: 1_000_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("2.08")), cost.String())
}

/*
OpenAI counts cached tokens inside input_tokens, so they were charged at the
full input price. The cached part is taken out of input and priced on its own.
*/
func TestProvider_CostFor_PricesOpenAICacheInsideInput(t *testing.T) {
	t.Parallel()

	for _, kind := range []Kind{KindOpenAIResponses, KindOpenAIChat} {
		luna := &Provider{
			Kind:                 kind,
			InputCostPerMillion:  money("2"),
			OutputCostPerMillion: money("8"),
		}
		cost := luna.CostFor(TokenUsage{Input: 1_000_000, Output: 0, CacheRead: 800_000})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.56")), "%s: %s", kind, cost)

		luna.CacheReadCostPerMillion = money("0.5")
		cost = luna.CostFor(TokenUsage{Input: 1_000_000, CacheRead: 800_000})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.8")), "%s: %s", kind, cost)

		cost = luna.CostFor(TokenUsage{Input: 100, CacheRead: 500})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.00025")),
			"a cached count above the input count never prices input below zero: %s", cost)
	}
}

func TestProvider_CachePrices(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (&Provider{Kind: KindAnthropicMessages}).CacheReadPrice(),
		"no input price, no cache price to derive")

	anthropic := &Provider{Kind: KindAnthropicMessages, InputCostPerMillion: money("3")}
	assert.True(t, anthropic.CacheReadPrice().Equal(decimal.RequireFromString("0.3")))
	assert.True(t, anthropic.CacheWritePrice().Equal(decimal.RequireFromString("3.75")))

	chat := &Provider{Kind: KindOpenAIChat, InputCostPerMillion: money("3")}
	assert.True(t, chat.CacheWritePrice().Equal(decimal.RequireFromString("3")),
		"a protocol that reports no writes prices one as input")

	assert.True(t, KindAnthropicMessages.CachesOutsideInput())
	assert.False(t, KindOpenAIResponses.CachesOutsideInput())
	assert.False(t, KindOpenAIChat.CachesOutsideInput())
	assert.False(t, KindOllama.CachesOutsideInput())
}
