package benchrecord

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
)

func TestUsageOf_CountsTheAdvisorApart(t *testing.T) {
	t.Parallel()

	call := &ModelCall{ChatResult: &llmports.ChatCompletionResult{
		ModelIdentifier: "claude-haiku-5-5",
		InputTokens:     100,
		OutputTokens:    20,
		ProviderKind:    aiprovider.KindAnthropicMessages,
		Advisor: &aipricing.AdvisorUsage{
			Model:  "claude-opus-5-5",
			Calls:  2,
			Tokens: aipricing.TokenUsage{Input: 300, CacheRead: 50, Output: 40},
		},
	}}

	usage, _ := UsageOf([]*ModelCall{call}, nil)
	assert.Equal(t, 100, usage.InputTokens)
	assert.Equal(t, 20, usage.OutputTokens)
	assert.Equal(t, 2, usage.AdvisorCalls)
	assert.Equal(t, 350, usage.AdvisorPromptTokens)
	assert.Equal(t, 40, usage.AdvisorOutputTokens)
	assert.Equal(t, "claude-opus-5-5", summarizeCall(call, nil).Advisor.Model)
}
