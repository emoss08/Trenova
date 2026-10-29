package aipricing_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pricedProvider(model, input, output string) *aiprovider.Provider {
	in, out := decimal.RequireFromString(input), decimal.RequireFromString(output)

	return &aiprovider.Provider{
		Kind:                 aiprovider.KindAnthropicMessages,
		Model:                model,
		InputCostPerMillion:  &in,
		OutputCostPerMillion: &out,
	}
}

func assertCost(t *testing.T, want string, got *decimal.Decimal, msg string) {
	t.Helper()

	require.NotNil(t, got, msg)
	assert.True(t, got.Equal(decimal.RequireFromString(want)), "%s: got %s", msg, got.String())
}

func TestProvider_AdvisedCostFor(t *testing.T) {
	t.Parallel()

	executor := pricedProvider("house-model", "0.5", "0")
	usage := aipricing.TokenUsage{Input: 1_000_000}
	advisor := &aipricing.AdvisorUsage{
		Model: "claude-opus-5-5",
		Calls: 1,
		Tokens: aipricing.TokenUsage{
			Input: 1_000_000, CacheRead: 1_000_000, CacheWrite: 1_000_000, Output: 1_000_000,
		},
	}

	assertCost(t, "29.7", aipricing.AdvisedCostFor(executor, usage, advisor),
		"$4 input, $0.20 cache read, $5 cache write and $20 output, on top of the executor")
	assertCost(
		t,
		"0.5",
		aipricing.AdvisedCostFor(executor, usage, nil),
		"no advisor, the executor alone",
	)
	assert.Nil(t, aipricing.AdvisedCostFor(
		&aiprovider.Provider{Kind: aiprovider.KindAnthropicMessages}, usage, advisor),
		"an unpriced executor stays unpriced")
	assert.Nil(t, aipricing.AdvisedCostFor(executor, usage,
		&aipricing.AdvisorUsage{Model: "house-advisor", Calls: 1}),
		"an advisor without a list price leaves the call unpriced rather than undercounted")
	assert.False(t, (&aipricing.AdvisorUsage{Model: "claude-opus-5-5"}).Used())

	assertCost(
		t,
		"0.5",
		aipricing.AdvisedCostForTask(executor, aitask.TaskEmbedding, usage, advisor),
		"a task that generates nothing is priced on its input alone",
	)
}

func TestProvider_AdvisedCostFor_PricesEachExecutorIterationOnItsOwnTier(t *testing.T) {
	t.Parallel()

	haiku := pricedProvider("claude-haiku-5-5", "0.10", "0.50")
	summed := aipricing.TokenUsage{Input: 120_000, Output: 2_000}
	advisor := &aipricing.AdvisorUsage{
		Model:  "claude-opus-5-5",
		Calls:  1,
		Tokens: aipricing.TokenUsage{Input: 10_000, Output: 1_000},
		Executor: []aipricing.TokenUsage{
			{Input: 60_000, Output: 1_000},
			{Input: 60_000, Output: 1_000},
		},
	}

	assertCost(t, "0.065", aipricing.CostFor(haiku, summed),
		"the summed prompt alone would cross Haiku 5.5's long-prompt line")
	assertCost(
		t,
		"0.073",
		aipricing.AdvisedCostFor(haiku, summed, advisor),
		"two 60K-prompt iterations at the short rates ($0.013) plus the advisor at Opus 5.5's ($0.06)",
	)

	long := *advisor
	long.Executor = []aipricing.TokenUsage{
		{Input: 110_000, Output: 1_000},
		{Input: 10_000, Output: 1_000},
	}
	assertCost(t, "0.119", aipricing.AdvisedCostFor(haiku, summed, &long),
		"only the iteration over the line is priced at the long rates")

	unsplit := *advisor
	unsplit.Executor = []aipricing.TokenUsage{{Input: 60_000, Output: 1_000}}
	assertCost(t, "0.125", aipricing.AdvisedCostFor(haiku, summed, &unsplit),
		"iterations that do not add up to the reported usage price the reported usage")
}

func TestPlatform_ServesAdvisorOnlyOnTheClaudeAPIAndClaudePlatformOnAWS(t *testing.T) {
	t.Parallel()

	served := make([]aiplatform.Platform, 0, 2)
	for _, platform := range aiplatform.AllPlatforms() {
		if platform.ServesAdvisor() {
			served = append(served, platform)
		}
	}
	assert.Equal(t, []aiplatform.Platform{
		aiplatform.PlatformDirect,
		aiplatform.PlatformClaudeAWS,
	}, served)
}
