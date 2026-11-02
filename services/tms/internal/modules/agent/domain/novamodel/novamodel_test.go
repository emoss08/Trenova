package novamodel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/novamodel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deskProvider() *aiprovider.Provider {
	input, output := decimal.NewFromInt(4), decimal.NewFromInt(20)
	cacheRead := decimal.RequireFromString("0.20")
	window := 1_000_000
	limit := decimal.NewFromInt(500)

	return &aiprovider.Provider{
		ID:                      pulid.MustNew("aiprv_"),
		Name:                    "Anthropic",
		Kind:                    aiprovider.KindAnthropicMessages,
		BaseURL:                 "https://api.anthropic.com",
		Model:                   "claude-opus-5-5",
		APIKey:                  "sealed",
		ReasoningEffort:         aireasoning.ReasoningHigh,
		ThinkingStyle:           aireasoning.ThinkingStyleEffort,
		ContextWindowTokens:     &window,
		AdvisorEnabled:          true,
		AdvisorModel:            "claude-fable-5-1",
		InputCostPerMillion:     &input,
		OutputCostPerMillion:    &output,
		CacheReadCostPerMillion: &cacheRead,
		Tasks:                   []aitask.Task{aitask.TaskAssistantChat},
		TimeoutSeconds:          90,
		MaxConcurrent:           4,
		MonthlyCapUSD:           &limit,
		Enabled:                 true,
		Nova:                    true,
	}
}

func TestProvider_CallsTheProviderWithNovasModelReasoningAndPrice(t *testing.T) {
	t.Parallel()

	desk := deskProvider()
	nova := novamodel.Provider(desk)

	assert.Equal(t, "claude-haiku-5-5", nova.Model)
	assert.Equal(t, aireasoning.ReasoningOff, nova.ReasoningEffort)
	assert.Equal(t, aireasoning.ThinkingStyleAuto, nova.ThinkingStyle)
	assert.Nil(t, nova.ContextWindowTokens, "the window is read off Nova's model id")
	assert.False(t, nova.AdvisorEnabled)
	assert.Empty(t, nova.AdvisorModel)

	card, ok := aipricing.ListPrice(aiprovider.KindAnthropicMessages, novamodel.ID)
	require.True(t, ok, "Nova's model has a list price")
	require.NotNil(t, nova.InputCostPerMillion)
	require.NotNil(t, nova.OutputCostPerMillion)
	assert.True(t, card.InputCostPerMillion.Equal(*nova.InputCostPerMillion))
	assert.True(t, card.OutputCostPerMillion.Equal(*nova.OutputCostPerMillion))
	assert.Nil(t, nova.CacheReadCostPerMillion, "the cache prices follow Nova's list price")
	assert.Nil(t, nova.CacheWriteCostPerMillion)
}

func TestProvider_KeepsTheProvidersEndpointCredentialAndLimits(t *testing.T) {
	t.Parallel()

	desk := deskProvider()
	nova := novamodel.Provider(desk)

	assert.Equal(t, desk.ID, nova.ID, "health, slots and spend are the provider's own")
	assert.Equal(t, desk.BaseURL, nova.BaseURL)
	assert.Equal(t, desk.APIKey, nova.APIKey)
	assert.Equal(t, desk.TimeoutSeconds, nova.TimeoutSeconds)
	assert.Equal(t, desk.MaxConcurrent, nova.MaxConcurrent)
	assert.Equal(t, desk.MonthlyCapUSD, nova.MonthlyCapUSD)
}

func TestProvider_LeavesTheStoredProviderAsItIs(t *testing.T) {
	t.Parallel()

	desk := deskProvider()
	_ = novamodel.Provider(desk)

	assert.Equal(t, "claude-opus-5-5", desk.Model)
	assert.True(t, desk.AdvisorEnabled)
	assert.True(t, desk.InputCostPerMillion.Equal(decimal.NewFromInt(4)))
}
