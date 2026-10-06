package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
)

/*
OpenAI caches a prompt's prefix automatically but spreads requests across
machines, and a cache key keeps the requests that share a prefix together.
It is sent only to OpenAI itself: a server speaking the same protocol elsewhere
may refuse a field it does not know.
*/
func TestOpenAIResponses_SendsTheCacheKeyOnlyToOpenAI(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		baseURL string
		want    string
	}{
		{"https://api.openai.com", "agent-key"},
		{"", "agent-key"},
		{"https://bedrock-mantle.us-east-1.api.aws", ""},
	} {
		call := callFor(aiprovider.KindOpenAIResponses, tc.baseURL, &Request{
			Messages: UserMessage("hi"),
			CacheKey: "agent-key",
		})
		call.Provider.BaseURL = tc.baseURL

		adapter := openAIResponsesAdapter{}
		assert.Equal(t, tc.want, adapter.requestFor(call).PromptCacheKey, tc.baseURL)
		assert.Equal(t, tc.want, adapter.streamRequestFor(call).PromptCacheKey, tc.baseURL)
	}
}
