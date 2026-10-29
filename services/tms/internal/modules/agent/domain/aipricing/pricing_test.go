package aipricing

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
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

	unpriced := &aiprovider.Provider{}
	assert.False(t, unpriced.Priced())
	assert.Nil(t, CostFor(unpriced, TokenUsage{Input: 1000, Output: 1000}))

	half := &aiprovider.Provider{InputCostPerMillion: money("3")}
	assert.False(t, half.Priced(), "one price is not a price list")
	assert.Nil(t, CostFor(half, TokenUsage{Input: 1000, Output: 1000}))

	claude := &aiprovider.Provider{
		InputCostPerMillion:  money("3"),
		OutputCostPerMillion: money("15"),
	}
	cost := CostFor(claude, TokenUsage{Input: 2_000_000, Output: 100_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("7.5")), cost.String())

	free := &aiprovider.Provider{InputCostPerMillion: money("0"), OutputCostPerMillion: money("0")}
	cost = CostFor(free, TokenUsage{Input: 5000, Output: 5000})
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

	haiku := &aiprovider.Provider{
		Kind:                 aiprovider.KindAnthropicMessages,
		InputCostPerMillion:  money("1"),
		OutputCostPerMillion: money("5"),
	}
	cost := CostFor(haiku, TokenUsage{
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
	cost = CostFor(haiku, TokenUsage{CacheRead: 1_000_000, CacheWrite: 1_000_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("2.08")), cost.String())
}

/*
OpenAI counts cached tokens inside input_tokens, so they were charged at the
full input price. The cached part is taken out of input and priced on its own.
*/
func TestProvider_CostFor_PricesOpenAICacheInsideInput(t *testing.T) {
	t.Parallel()

	for _, kind := range []aiprovider.Kind{aiprovider.KindOpenAIResponses, aiprovider.KindOpenAIChat} {
		luna := &aiprovider.Provider{
			Kind:                 kind,
			InputCostPerMillion:  money("2"),
			OutputCostPerMillion: money("8"),
		}
		cost := CostFor(luna, TokenUsage{Input: 1_000_000, Output: 0, CacheRead: 800_000})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.56")), "%s: %s", kind, cost)

		luna.CacheReadCostPerMillion = money("0.5")
		cost = CostFor(luna, TokenUsage{Input: 1_000_000, CacheRead: 800_000})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.8")), "%s: %s", kind, cost)

		cost = CostFor(luna, TokenUsage{Input: 100, CacheRead: 500})
		require.NotNil(t, cost)
		assert.True(t, cost.Equal(decimal.RequireFromString("0.00025")),
			"a cached count above the input count never prices input below zero: %s", cost)
	}
}

func TestProvider_CachePrices(t *testing.T) {
	t.Parallel()

	assert.Nil(t, CacheReadPrice(&aiprovider.Provider{Kind: aiprovider.KindAnthropicMessages}),
		"no input price, no cache price to derive")

	anthropic := &aiprovider.Provider{
		Kind:                aiprovider.KindAnthropicMessages,
		InputCostPerMillion: money("3"),
	}
	assert.True(t, CacheReadPrice(anthropic).Equal(decimal.RequireFromString("0.3")))
	assert.True(t, CacheWritePrice(anthropic).Equal(decimal.RequireFromString("3.75")))

	chat := &aiprovider.Provider{Kind: aiprovider.KindOpenAIChat, InputCostPerMillion: money("3")}
	assert.True(t, CacheWritePrice(chat).Equal(decimal.RequireFromString("3")),
		"a protocol that reports no writes prices one as input")

	assert.True(t, aiprovider.KindAnthropicMessages.CachesOutsideInput())
	assert.False(t, aiprovider.KindOpenAIResponses.CachesOutsideInput())
	assert.False(t, aiprovider.KindOpenAIChat.CachesOutsideInput())
}

func TestListPrice(t *testing.T) {
	t.Parallel()

	_, ok := ListPrice(aiprovider.KindOpenAIResponses, "claude-haiku-5-5")
	assert.False(t, ok)
	_, ok = ListPrice(aiprovider.KindAnthropicMessages, "claude-unknown-9")
	assert.False(t, ok)

	opus, ok := ListPrice(aiprovider.KindAnthropicMessages, "anthropic.claude-opus-5-5")
	require.True(t, ok)
	assert.True(t, opus.CacheReadCostPerMillion().Equal(decimal.RequireFromString("0.2")))
	assert.True(t, opus.CacheWriteCostPerMillion().Equal(decimal.RequireFromString("5")))

	fable, ok := ListPrice(aiprovider.KindAnthropicMessages, "claude-fable-5-1")
	require.True(t, ok)
	assert.True(t, fable.CacheReadCostPerMillion().Equal(decimal.RequireFromString("0.25")))

	haiku, ok := ListPrice(aiprovider.KindAnthropicMessages, "claude-haiku-5-5")
	require.True(t, ok)
	assert.True(t, haiku.InputCostPerMillion.Equal(decimal.RequireFromString("0.1")))
	assert.True(t, haiku.CacheReadCostPerMillion().Equal(decimal.RequireFromString("0.01")))
	require.NotNil(t, haiku.LongPrompt)
	assert.Equal(t, 100_000, haiku.LongPrompt.AbovePromptTokens)
}

func TestProvider_CostFor_HaikuLongPromptTier(t *testing.T) {
	t.Parallel()

	haiku := &aiprovider.Provider{
		Kind:                 aiprovider.KindAnthropicMessages,
		Model:                "claude-haiku-5-5",
		InputCostPerMillion:  money("0.10"),
		OutputCostPerMillion: money("0.50"),
	}

	short := CostFor(haiku, TokenUsage{Input: 10_000, CacheRead: 90_000, Output: 1_000_000})
	require.NotNil(t, short)
	want := decimal.RequireFromString("0.001").
		Add(decimal.RequireFromString("0.0009")).
		Add(decimal.RequireFromString("0.5"))
	assert.True(t, short.Equal(want), short.String())

	long := CostFor(haiku, TokenUsage{Input: 10_000, CacheRead: 90_001, Output: 1_000_000})
	require.NotNil(t, long)
	want = decimal.RequireFromString("0.005").
		Add(decimal.RequireFromString("0.00450005")).
		Add(decimal.RequireFromString("2.5"))
	assert.True(t, long.Equal(want), long.String())

	write := CostFor(haiku, TokenUsage{CacheWrite: 1_000_000})
	require.NotNil(t, write)
	assert.True(t, write.Equal(decimal.RequireFromString("0.625")), write.String())
}

func TestProvider_CostFor_OpusCacheReadAtItsOwnRate(t *testing.T) {
	t.Parallel()

	opus := &aiprovider.Provider{
		Kind:                 aiprovider.KindAnthropicMessages,
		Model:                "claude-opus-5-5",
		InputCostPerMillion:  money("4"),
		OutputCostPerMillion: money("20"),
	}
	cost := CostFor(opus, TokenUsage{CacheRead: 1_000_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("0.2")), cost.String())

	opus.CacheReadCostPerMillion = money("0.3")
	cost = CostFor(opus, TokenUsage{CacheRead: 1_000_000})
	require.NotNil(t, cost)
	assert.True(t, cost.Equal(decimal.RequireFromString("0.3")), cost.String())
}

func TestTokenUsage_PromptTokens(t *testing.T) {
	t.Parallel()

	usage := TokenUsage{Input: 100, CacheRead: 50, CacheWrite: 25}
	assert.Equal(t, 175, usage.PromptTokens(aiprovider.KindAnthropicMessages))
	assert.Equal(t, 100, usage.PromptTokens(aiprovider.KindOpenAIResponses))
}

func TestProvider_BatchCostForTask_HalvesEveryTokenPrice(t *testing.T) {
	t.Parallel()

	sonnet := &aiprovider.Provider{
		Kind:                 aiprovider.KindAnthropicMessages,
		Model:                "claude-sonnet-5-5",
		InputCostPerMillion:  money("2"),
		OutputCostPerMillion: money("10"),
	}
	usage := TokenUsage{Input: 1_000_000, Output: 100_000, CacheRead: 1_000_000}

	full := CostForTask(sonnet, aitask.TaskEvaluationJudge, usage)
	batched := BatchCostForTask(sonnet, aitask.TaskEvaluationJudge, usage)
	require.NotNil(t, full)
	require.NotNil(t, batched)
	assert.True(t, batched.Equal(full.Div(decimal.NewFromInt(2))), batched.String())

	assert.Nil(t, BatchCostForTask(&aiprovider.Provider{}, aitask.TaskEvaluationJudge, usage),
		"an unpriced provider stays unpriced in a batch")
}
