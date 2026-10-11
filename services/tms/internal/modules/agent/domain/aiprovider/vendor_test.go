package aiprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderVendor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		provider Provider
		want     string
	}{
		{"anthropic default", Provider{Kind: KindAnthropicMessages}, "anthropic"},
		{"openai default", Provider{Kind: KindOpenAIResponses}, "openai"},
		{"groq chat", Provider{Kind: KindOpenAIChat, BaseURL: "https://api.groq.com/openai/v1"}, "groq"},
		{"gemini chat", Provider{Kind: KindOpenAIChat, BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai"}, "gemini"},
		{"mistral chat", Provider{Kind: KindOpenAIChat, BaseURL: "https://api.mistral.ai/v1"}, "mistral"},
		{"ollama", Provider{Kind: KindOllama}, "ollama"},
		{"self hosted", Provider{Kind: KindOpenAIChat, BaseURL: "http://10.0.0.4:8000/v1"}, ""},
		{"lookalike host", Provider{Kind: KindOpenAIChat, BaseURL: "https://notgroq.com/v1"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.provider.Vendor())
		})
	}
}

func TestProviderFailedLastTest(t *testing.T) {
	t.Parallel()

	assert.False(t, (&Provider{}).FailedLastTest())
	assert.False(t, (&Provider{LastTest: &TestOutcome{Success: true}}).FailedLastTest())
	assert.True(t, (&Provider{LastTest: &TestOutcome{Success: false}}).FailedLastTest())
}
