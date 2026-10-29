package aiproviderservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceIndex_CarriesCachePrices(t *testing.T) {
	t.Parallel()

	index := newPriceIndex(map[string]llmports.CachedModelPrice{
		"openai/gpt-6-luna":  {Input: "1", Output: "8", CacheRead: "0.1", CacheWrite: "x"},
		"openai/gpt-6.1-sol": {Input: "2", Output: "10"},
	})

	luna, ok := index.lookup(aiprovider.KindOpenAIResponses, "gpt-6-luna")
	require.True(t, ok)
	require.NotNil(t, luna.CacheReadCostPerMillion)
	assert.Equal(t, "0.1", luna.CacheReadCostPerMillion.String())
	assert.Nil(t, luna.CacheWriteCostPerMillion, "an unreadable price is no price")

	sol, ok := index.lookup(aiprovider.KindOpenAIResponses, "gpt-6.1-sol")
	require.True(t, ok)
	assert.Nil(t, sol.CacheReadCostPerMillion)
	require.NotNil(t, sol.OutputCostPerMillion)
	assert.Equal(t, "10", sol.OutputCostPerMillion.String())
}
