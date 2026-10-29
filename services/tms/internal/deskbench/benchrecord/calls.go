package benchrecord

import (
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const digestLength = 12

func DigestOf(value any) string {
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return ""
	}
	return hashutils.SHA256BytesHex(encoded)[:digestLength]
}

type CallPricing func(call *ModelCall) CallPrice

func SummarizeCalls(calls []*ModelCall, priceOf CallPricing) []CallSummary {
	summaries := make([]CallSummary, 0, len(calls))
	for _, call := range calls {
		summaries = append(summaries, summarizeCall(call, priceOf))
	}

	return summaries
}

func summarizeCall(call *ModelCall, priceOf CallPricing) CallSummary {
	summary := CallSummary{
		Seq:       call.Seq,
		Kind:      call.Kind,
		Delegate:  call.Attribution.DelegateCallID,
		Retries:   call.Retries,
		Error:     call.Error,
		LatencyMs: call.Duration().Milliseconds(),
	}
	if !call.FirstTokenAt.IsZero() {
		first := call.FirstTokenAt.Sub(call.StartedAt).Milliseconds()
		summary.FirstTokenMs = &first
	}

	if req := call.Chat; req != nil {
		summary.Messages = len(req.Messages)
		summary.Tools = len(req.Tools)
		summary.SystemDigest = DigestOf(req.System)
		summary.ToolsDigest = DigestOf(req.Tools)
		summary.Reask = req.Reask
	}
	if req := call.Structured; req != nil {
		summary.SystemDigest = DigestOf(req.System)
	}

	tokens := CallTokens(call)
	summary.InputTokens = tokens.uncached
	summary.CacheRead = tokens.read
	summary.CacheWrite = tokens.write
	summary.OutputTokens = tokens.output
	summary.ReasoningTokens = tokens.reasoning
	summary.Advisor = tokens.advisor
	if priceOf != nil {
		price := priceOf(call)
		summary.CostUSD = price.cost
		summary.PriceSource = price.source
	}

	if result := call.ChatResult; result != nil {
		summary.Model = result.ModelIdentifier
		summary.Provider = string(result.ProviderKind)
		summary.Truncated = result.Truncated
		summary.Text = result.Text
		summary.ToolCalls = result.ToolCalls
		summary.FallbackFrom = result.FallbackFrom
		if result.LatencyMs > 0 {
			summary.LatencyMs = result.LatencyMs
		}
		if result.CutOffCall != nil {
			summary.CutOffCall = result.CutOffCall.Name
			if summary.CutOffCall == "" {
				summary.CutOffCall = "(unnamed)"
			}
		}
		if result.Reasoning != nil {
			summary.Thinking = result.Reasoning.Text
		}
	}
	if result := call.StructuredResult; result != nil {
		summary.Model = result.ModelIdentifier
		summary.Provider = string(result.ProviderKind)
		summary.Text = result.Text
		if result.LatencyMs > 0 {
			summary.LatencyMs = result.LatencyMs
		}
	}

	return summary
}

type tokenCount struct {
	uncached  int
	read      int
	write     int
	output    int
	reasoning int
	raw       aipricing.TokenUsage
	kind      aiprovider.Kind
	model     string
	provider  pulid.ID
	charged   *decimal.Decimal
	advisor   *aipricing.AdvisorUsage
}

func CallTokens(call *ModelCall) tokenCount {
	count := tokenCount{}
	switch {
	case call.ChatResult != nil:
		result := call.ChatResult
		count.raw = aipricing.TokenUsage{
			Input:      result.InputTokens,
			Output:     result.OutputTokens,
			CacheRead:  result.CacheReadTokens,
			CacheWrite: result.CacheWriteTokens,
		}
		count.reasoning = result.ReasoningTokens
		count.kind = result.ProviderKind
		count.model = result.ModelIdentifier
		count.provider = result.ProviderID
		count.charged = result.CostUSD
		count.advisor = result.Advisor
	case call.StructuredResult != nil:
		result := call.StructuredResult
		count.raw = aipricing.TokenUsage{
			Input:      result.InputTokens,
			Output:     result.OutputTokens,
			CacheRead:  result.CacheReadTokens,
			CacheWrite: result.CacheWriteTokens,
		}
		count.kind = result.ProviderKind
		count.model = result.ModelIdentifier
		count.provider = result.ProviderID
		count.charged = result.CostUSD
	default:
		return count
	}

	count.read = max(count.raw.CacheRead, 0)
	count.write = max(count.raw.CacheWrite, 0)
	count.output = max(count.raw.Output, 0)
	count.uncached = max(count.raw.Input, 0)
	if !count.kind.CachesOutsideInput() {
		count.uncached = max(count.uncached-count.read-count.write, 0)
	}

	return count
}

func UsageOf(calls []*ModelCall, priceOf CallPricing) (usage Usage, models []string) {
	models = make([]string, 0, 2)
	for _, call := range calls {
		usage.ModelCalls++
		usage.ModelTime += call.Duration()

		tokens := CallTokens(call)
		if tokens.model == "" && tokens.raw == (aipricing.TokenUsage{}) {
			continue
		}
		next := Usage{
			InputTokens:      tokens.uncached,
			CacheReadTokens:  tokens.read,
			CacheWriteTokens: tokens.write,
			OutputTokens:     tokens.output,
			ReasoningTokens:  tokens.reasoning,
		}
		if advisor := tokens.advisor; advisor.Used() {
			next.AdvisorCalls = advisor.Calls
			next.AdvisorPromptTokens = advisor.Tokens.Input + advisor.Tokens.CacheRead +
				advisor.Tokens.CacheWrite
			next.AdvisorOutputTokens = advisor.Tokens.Output
		}
		price := CallPrice{cost: tokens.charged, source: PriceSourceProvider}
		if priceOf != nil {
			price = priceOf(call)
		}
		switch {
		case price.cost == nil:
			next.UnpricedCalls = 1
		case price.source != PriceSourceProvider:
			next.EstimatedCalls = 1
		}
		next.CostUSD = price.cost
		usage.Add(next)
		if tokens.model != "" && !slices.Contains(models, tokens.model) {
			models = append(models, tokens.model)
		}
	}

	return usage, models
}
