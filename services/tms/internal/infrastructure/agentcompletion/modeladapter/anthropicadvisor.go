package modeladapter

import (
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

const anthropicAdvisorBeta = anthropic.AnthropicBetaAdvisorTool2026_03_01

const (
	anthropicServerToolUseBlock = conversation.ProviderBlockServerToolUse
	anthropicAdvisorResultBlock = conversation.ProviderBlockAdvisorToolResult
	anthropicAdvisorIteration   = "advisor_message"
	anthropicExecutorIteration  = "message"
	anthropicAdvisorResult      = "advisor_result"
	anthropicAdvisorRedacted    = "advisor_redacted_result"
)

func servesAdvisor(call *Call) bool {
	return call.Request != nil && call.Provider != nil &&
		call.Provider.ResolvedPlatform().ServesAdvisor()
}

func advisorFor(call *Call) *AdvisorSpec {
	if !servesAdvisor(call) {
		return nil
	}
	if spec := call.Request.Advisor; spec != nil && spec.Model != "" {
		return spec
	}

	return nil
}

func sendsAdvisorBeta(call *Call, advisor *AdvisorSpec) bool {
	if !servesAdvisor(call) {
		return false
	}

	return advisor != nil || replaysAdvice(call.Request.Messages)
}

func replaysAdvice(messages []Message) bool {
	for idx := range messages {
		msg := &messages[idx]
		if msg.Role == RoleAssistant &&
			msg.Reasoning.ReplayableBy(string(aiprovider.KindAnthropicMessages)) &&
			msg.Reasoning.HoldsServerTools() {
			return true
		}
	}

	return false
}

func withoutAdvice(messages []Message) []Message {
	var out []Message
	for idx := range messages {
		trace := messages[idx].Reasoning.WithoutServerTools()
		if trace == messages[idx].Reasoning {
			continue
		}
		if out == nil {
			out = slices.Clone(messages)
		}
		out[idx].Reasoning = trace
	}
	if out == nil {
		return messages
	}

	return out
}

func advisedTools(
	tools []anthropic.BetaToolUnionParam,
	spec *AdvisorSpec,
) []anthropic.BetaToolUnionParam {
	if spec == nil {
		return tools
	}
	tool := &anthropic.BetaAdvisorTool20260301Param{
		Model: spec.Model,
		Caching: anthropic.BetaCacheControlEphemeralParam{
			TTL: anthropic.BetaCacheControlEphemeralTTLTTL5m,
		},
	}
	if spec.MaxUses > 0 {
		tool.MaxUses = param.NewOpt(int64(spec.MaxUses))
	}
	if spec.MaxTokens > 0 {
		tool.MaxTokens = param.NewOpt(int64(spec.MaxTokens))
	}

	return append(tools, anthropic.BetaToolUnionParam{OfAdvisorTool20260301: tool})
}

func anthropicAdvisorUsage(
	iterations anthropic.BetaIterationsUsage,
	spec *AdvisorSpec,
) *aiprovider.AdvisorUsage {
	usage := aiprovider.AdvisorUsage{}
	if spec != nil {
		usage.Model = spec.Model
	}
	executor := make([]aiprovider.TokenUsage, 0, len(iterations))
	for idx := range iterations {
		iteration := &iterations[idx]
		tokens := aiprovider.TokenUsage{
			Input:      int(iteration.InputTokens),
			Output:     int(iteration.OutputTokens),
			CacheRead:  int(iteration.CacheReadInputTokens),
			CacheWrite: int(iteration.CacheCreationInputTokens),
		}
		switch iteration.Type {
		case anthropicExecutorIteration:
			executor = append(executor, tokens)
		case anthropicAdvisorIteration:
			usage.Calls++
			if iteration.Model != "" {
				usage.Model = iteration.Model
			}
			usage.Tokens.Input += tokens.Input
			usage.Tokens.Output += tokens.Output
			usage.Tokens.CacheRead += tokens.CacheRead
			usage.Tokens.CacheWrite += tokens.CacheWrite
		}
	}
	if !usage.Used() {
		return nil
	}
	if len(executor) > 0 {
		usage.Executor = executor
	}

	return &usage
}

func advisorModelOf(usage *aiprovider.AdvisorUsage, spec *AdvisorSpec) string {
	if usage != nil && usage.Model != "" {
		return usage.Model
	}
	if spec != nil {
		return spec.Model
	}

	return ""
}

func providerBlocksOf(blocks []anthropicBlock, advisorModel string) []ProviderBlock {
	if !holdsServerTool(blocks) {
		return nil
	}
	stored := make([]ProviderBlock, 0, len(blocks))
	for idx := range blocks {
		block := &blocks[idx]
		switch block.Type {
		case anthropicTextBlock:
			stored = append(stored, ProviderBlock{Type: block.Type, Text: block.Text})
		case anthropicThinkingBlock:
			stored = append(stored, ProviderBlock{
				Type:      block.Type,
				Text:      block.Thinking,
				Signature: block.Signature,
			})
		case anthropicRedactedBlock:
			stored = append(stored, ProviderBlock{Type: block.Type, Data: block.Data})
		case anthropicServerToolUseBlock:
			stored = append(stored, ProviderBlock{
				Type:  block.Type,
				ID:    block.ID,
				Name:  block.Name,
				Input: toolUseInput(block.Arguments),
			})
		case anthropicAdvisorResultBlock:
			stored = append(stored, ProviderBlock{
				Type:      block.Type,
				ToolUseID: block.ToolUseID,
				Result:    block.Result,
				Model:     advisorModel,
			})
		}
	}

	return stored
}

func holdsServerTool(blocks []anthropicBlock) bool {
	for idx := range blocks {
		switch blocks[idx].Type {
		case anthropicServerToolUseBlock, anthropicAdvisorResultBlock:
			return true
		}
	}

	return false
}

func storedBlocks(trace *ReasoningTrace) []ProviderBlock {
	if !trace.ReplayableBy(string(aiprovider.KindAnthropicMessages)) {
		return nil
	}

	return trace.Blocks
}

func replayProviderBlocks(
	stored []ProviderBlock,
	content string,
	current bool,
) []anthropic.BetaContentBlockParamUnion {
	replace := replacesStoredText(stored, content)
	out := make([]anthropic.BetaContentBlockParamUnion, 0, len(stored)+1)
	for idx := range stored {
		block := &stored[idx]
		switch block.Type {
		case anthropicThinkingBlock:
			if current && block.Signature != "" {
				out = append(out, anthropic.BetaContentBlockParamUnion{
					OfThinking: &anthropic.BetaThinkingBlockParam{
						Thinking:  block.Text,
						Signature: block.Signature,
					},
				})
			}
		case anthropicRedactedBlock:
			if current {
				out = append(out, anthropic.BetaContentBlockParamUnion{
					OfRedactedThinking: &anthropic.BetaRedactedThinkingBlockParam{Data: block.Data},
				})
			}
		case anthropicTextBlock:
			if replace || strings.TrimSpace(block.Text) == "" {
				continue
			}
			out = append(out, anthropic.BetaContentBlockParamUnion{
				OfText: &anthropic.BetaTextBlockParam{Text: block.Text},
			})
		case anthropicServerToolUseBlock:
			out = append(out, anthropic.BetaContentBlockParamUnion{
				OfServerToolUse: &anthropic.BetaServerToolUseBlockParam{
					ID:    block.ID,
					Name:  anthropic.BetaServerToolUseBlockParamName(block.Name),
					Input: toolUseInput(block.Input),
				},
			})
		case anthropicAdvisorResultBlock:
			out = append(out, anthropic.BetaContentBlockParamUnion{
				OfAdvisorToolResult: &anthropic.BetaAdvisorToolResultBlockParam{
					ToolUseID: block.ToolUseID,
					Content:   advisorResultParam(block.Result),
				},
			})
		}
	}
	if replace {
		out = append(out, anthropic.BetaContentBlockParamUnion{
			OfText: &anthropic.BetaTextBlockParam{Text: content},
		})
	}

	return out
}

func replacesStoredText(stored []ProviderBlock, content string) bool {
	kept := strings.TrimSpace(content)
	if kept == "" {
		return false
	}
	var storedText strings.Builder
	for idx := range stored {
		if stored[idx].Type == anthropicTextBlock {
			storedText.WriteString(stored[idx].Text)
		}
	}

	return strings.TrimSpace(storedText.String()) != kept
}

func advisorResultParam(
	result *conversation.ServerToolResult,
) anthropic.BetaAdvisorToolResultBlockParamContentUnion {
	if result == nil {
		result = &conversation.ServerToolResult{}
	}
	stop := param.Opt[string]{}
	if result.StopReason != "" {
		stop = param.NewOpt(result.StopReason)
	}
	switch result.Type {
	case anthropicAdvisorResult:
		return anthropic.BetaAdvisorToolResultBlockParamContentUnion{
			OfRequestAdvisorResultBlock: &anthropic.BetaAdvisorResultBlockParam{
				Text:       result.Text,
				StopReason: stop,
			},
		}
	case anthropicAdvisorRedacted:
		return anthropic.BetaAdvisorToolResultBlockParamContentUnion{
			OfRequestAdvisorRedactedResultBlock: &anthropic.BetaAdvisorRedactedResultBlockParam{
				EncryptedContent: result.EncryptedContent,
				StopReason:       stop,
			},
		}
	default:
		return anthropic.BetaAdvisorToolResultBlockParamContentUnion{
			OfRequestAdvisorToolResultError: &anthropic.BetaAdvisorToolResultErrorParam{
				ErrorCode: anthropic.BetaAdvisorToolResultErrorParamErrorCode(result.ErrorCode),
			},
		}
	}
}
