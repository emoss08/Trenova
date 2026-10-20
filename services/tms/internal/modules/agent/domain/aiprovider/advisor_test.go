package aiprovider_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithAdvisorCost(t *testing.T) {
	t.Parallel()

	executor := decimal.RequireFromString("0.5")
	advisor := &aiprovider.AdvisorUsage{
		Model: "claude-opus-5-5",
		Calls: 1,
		Tokens: aiprovider.TokenUsage{
			Input: 1_000_000, CacheRead: 1_000_000, CacheWrite: 1_000_000, Output: 1_000_000,
		},
	}

	total := aiprovider.WithAdvisorCost(&executor, aiprovider.KindAnthropicMessages, advisor)
	require.NotNil(t, total)
	assert.True(t, total.Equal(decimal.RequireFromString("29.7")), total.String(),
		"$4 input, $0.20 cache read, $5 cache write and $20 output, on top of the executor")

	assert.Equal(t, &executor,
		aiprovider.WithAdvisorCost(&executor, aiprovider.KindAnthropicMessages, nil))
	assert.Nil(t, aiprovider.WithAdvisorCost(nil, aiprovider.KindAnthropicMessages, advisor),
		"an unpriced executor stays unpriced")
	assert.Nil(t, aiprovider.WithAdvisorCost(&executor, aiprovider.KindAnthropicMessages,
		&aiprovider.AdvisorUsage{Model: "house-advisor", Calls: 1}),
		"an advisor without a list price leaves the call unpriced rather than undercounted")
	assert.False(t, (&aiprovider.AdvisorUsage{Model: "claude-opus-5-5"}).Used())
}

func TestPlatform_ServesAdvisorOnlyOnTheClaudeAPIAndClaudePlatformOnAWS(t *testing.T) {
	t.Parallel()

	served := make([]aiprovider.Platform, 0, 2)
	for _, platform := range aiprovider.AllPlatforms() {
		if platform.ServesAdvisor() {
			served = append(served, platform)
		}
	}
	assert.Equal(t, []aiprovider.Platform{
		aiprovider.PlatformDirect,
		aiprovider.PlatformClaudeAWS,
	}, served)
}
