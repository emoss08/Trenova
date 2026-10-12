package deskbench

import (
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/hashutils"
)

const digestLength = 12

func digestOf(value any) string {
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return ""
	}
	return hashutils.SHA256BytesHex(encoded)[:digestLength]
}

func summarizeCalls(calls []*ModelCall) []CallSummary {
	summaries := make([]CallSummary, 0, len(calls))
	for _, call := range calls {
		summaries = append(summaries, summarizeCall(call))
	}

	return summaries
}

func summarizeCall(call *ModelCall) CallSummary {
	summary := CallSummary{
		Seq:       call.Seq,
		Kind:      call.Kind,
		Delegate:  call.Attribution.DelegateCallID,
		Retries:   call.Retries,
		Error:     call.Error,
		LatencyMs: call.Duration().Milliseconds(),
	}

	if req := call.Chat; req != nil {
		summary.Messages = len(req.Messages)
		summary.Tools = len(req.Tools)
		summary.SystemDigest = digestOf(req.System)
		summary.ToolsDigest = digestOf(req.Tools)
	}
	if req := call.Structured; req != nil {
		summary.SystemDigest = digestOf(req.System)
	}

	if result := call.ChatResult; result != nil {
		summary.Model = result.ModelIdentifier
		summary.Provider = string(result.ProviderKind)
		summary.InputTokens = result.InputTokens
		summary.OutputTokens = result.OutputTokens
		summary.ReasoningTokens = result.ReasoningTokens
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
		summary.InputTokens = result.InputTokens
		summary.OutputTokens = result.OutputTokens
		summary.Text = result.Text
		if result.LatencyMs > 0 {
			summary.LatencyMs = result.LatencyMs
		}
	}

	return summary
}

func usageOf(calls []*ModelCall) (Usage, []string) {
	usage := Usage{}
	models := make([]string, 0, 2)
	for _, call := range calls {
		usage.ModelCalls++
		usage.ModelTime += call.Duration()

		model := ""
		switch {
		case call.ChatResult != nil:
			result := call.ChatResult
			model = result.ModelIdentifier
			usage.add(Usage{
				InputTokens:     result.InputTokens,
				OutputTokens:    result.OutputTokens,
				ReasoningTokens: result.ReasoningTokens,
				CostUSD:         result.CostUSD,
			})
		case call.StructuredResult != nil:
			result := call.StructuredResult
			model = result.ModelIdentifier
			usage.add(Usage{
				InputTokens:  result.InputTokens,
				OutputTokens: result.OutputTokens,
				CostUSD:      result.CostUSD,
			})
		}
		if model != "" && !slices.Contains(models, model) {
			models = append(models, model)
		}
	}

	return usage, models
}
