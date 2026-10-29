package modelstream

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// FallbackOf is the provider asked first, when another one gave the reply.
func FallbackOf(completion *llmports.ChatCompletionResult) *conversation.ProviderFallback {
	if completion == nil || completion.FallbackFrom == nil {
		return nil
	}
	from := completion.FallbackFrom

	return &conversation.ProviderFallback{
		ProviderID: from.ProviderID,
		Name:       from.Name,
		Model:      from.Model,
		Status:     from.Status,
	}
}

func CannedCompletion(
	attempt *llmports.ChatCompletionResult,
	text string,
) *llmports.ChatCompletionResult {
	canned := &llmports.ChatCompletionResult{Text: text}
	if attempt != nil {
		canned.ModelIdentifier = attempt.ModelIdentifier
		canned.ProviderID = attempt.ProviderID
		canned.ProviderKind = attempt.ProviderKind
		canned.InputTokens = attempt.InputTokens
		canned.OutputTokens = attempt.OutputTokens
		canned.CacheReadTokens = attempt.CacheReadTokens
		canned.CacheWriteTokens = attempt.CacheWriteTokens
		canned.LatencyMs = attempt.LatencyMs
		canned.CostUSD = attempt.CostUSD
		canned.ContextWindow = attempt.ContextWindow
	}

	return canned
}

// tagReasoning records which protocol produced a trace, so an adapter of
// another kind knows not to send it back as its own.
// tagToolCalls names the provider on each call it made, so what the
// provider attached to the call is replayed to it and to nobody else.
func TagToolCalls(completion *llmports.ChatCompletionResult) {
	for idx := range completion.ToolCalls {
		completion.ToolCalls[idx].ProviderID = completion.ProviderID
	}
}

func TagReasoning(completion *llmports.ChatCompletionResult) {
	if completion.Reasoning != nil && completion.Reasoning.ProviderKind == "" {
		completion.Reasoning.ProviderKind = string(completion.ProviderKind)
	}
}
