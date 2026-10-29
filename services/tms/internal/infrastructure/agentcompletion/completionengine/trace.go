package completionengine

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	finishReasonStop          = "stop"
	finishReasonLength        = "length"
	finishReasonToolCalls     = "tool_calls"
	finishReasonContentFilter = "content_filter"
)

type AttemptSpec struct {
	Operation   string
	Provider    *aiprovider.Provider
	Attempt     int
	Failover    bool
	MaxTokens   int
	Attribution aiusageports.AIUsageAttribution
}

func (s *Engine) StartAttempt(
	ctx context.Context,
	spec *AttemptSpec,
) (context.Context, trace.Span) {
	return aitrace.StartAttempt(ctx, &aitrace.AttemptSpec{
		Anchor:        aitrace.ForAttribution(&spec.Attribution),
		Operation:     spec.Operation,
		ProviderKind:  spec.Provider.Kind,
		ProviderID:    spec.Provider.ID,
		Model:         spec.Provider.Model,
		MaxTokens:     spec.MaxTokens,
		ServerAddress: serverAddress(spec.Provider),
		Attempt:       spec.Attempt,
		Feature:       string(spec.Attribution.Feature),
		Attrs:         []attribute.KeyValue{aitrace.AIFailover.Bool(spec.Failover)},
	})
}

func (s *Engine) SettleAttempt(ctx context.Context, span trace.Span, attempt *UsageAttempt) {
	defer span.End()

	provider := attempt.Provider
	cost := attemptCost(attempt)
	if provider != nil && s.Health != nil && attempt.Err != nil {
		if _, resting := s.Health.Resting(provider.ID); resting {
			aitrace.RecordResting(span)
		}
	}

	s.Record(ctx, attempt)

	usage := attemptUsage(attempt, cost)
	aitrace.RecordUsage(span, usage)
	if attempt.FirstToken > 0 {
		span.SetAttributes(aitrace.AIFirstTokenMs.Int64(attempt.FirstToken.Milliseconds()))
	}
	errorType := ClassifyError(attempt.Err)
	if attempt.Err != nil {
		aitrace.MarkFailed(span, errorType)
	}

	aitrace.TallyFrom(ctx).Attempt(&aitrace.AttemptTally{
		ProviderID:       provider.ID.String(),
		ProviderName:     aitrace.ProviderName(provider.Kind),
		Model:            usage.ResponseModel,
		Failover:         attempt.Failover,
		CostUSD:          cost,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
	})

	s.GenAI.RecordCall(ctx, &aimetrics.GenAICall{
		Operation:     attempt.Operation,
		Provider:      aitrace.ProviderName(provider.Kind),
		RequestModel:  provider.Model,
		ResponseModel: usage.ResponseModel,
		ServerAddress: serverAddress(provider),
		ErrorType:     errorType,
		InputTokens:   usage.InputTokens,
		OutputTokens:  usage.OutputTokens,
		Duration:      attempt.Latency,
		Task:          string(attempt.Task),
		FirstToken:    attempt.FirstToken,
	})
}

func attemptCost(attempt *UsageAttempt) *decimal.Decimal {
	if attempt.Outcome == nil || attempt.Provider == nil {
		return nil
	}

	return attempt.Outcome.Cost(attempt.Provider, attempt.Task)
}

func attemptUsage(attempt *UsageAttempt, cost *decimal.Decimal) *aitrace.Usage {
	usage := &aitrace.Usage{CostUSD: cost}
	outcome := attempt.Outcome
	if outcome == nil {
		return usage
	}

	input := int64(outcome.InputTokens)
	if attempt.Provider.Kind.CachesOutsideInput() {
		input += int64(outcome.CacheReadTokens + outcome.CacheWriteTokens)
	}
	usage.ResponseModel = outcome.Model
	usage.InputTokens = input
	usage.OutputTokens = int64(outcome.OutputTokens)
	usage.CacheReadTokens = int64(outcome.CacheReadTokens)
	usage.CacheWriteTokens = int64(outcome.CacheWriteTokens)
	usage.ReasoningTokens = int64(outcome.ReasoningTokens)
	usage.Truncated = outcome.Truncated
	usage.ThinkingDropped = int64(outcome.ThinkingDropped)
	if diagnosis := outcome.CacheDiagnosis; diagnosis != nil {
		usage.CacheMissReason = diagnosis.Reason
		usage.CacheMissedTokens = int64(diagnosis.MissedTokens)
	}
	if outcome.FinishReason != "" {
		usage.FinishReasons = []string{outcome.FinishReason}
	}
	if outcome.Advisor.Used() {
		usage.Advisor = outcome.Advisor
		usage.AdvisorCostUSD = outcome.Advisor.Cost(attempt.Provider.Kind)
	}

	return usage
}

func FinishReason(resp *modeladapter.Response) string {
	switch {
	case resp == nil:
		return ""
	case resp.Refused:
		return finishReasonContentFilter
	case resp.Truncated:
		return finishReasonLength
	case len(resp.ToolCalls) > 0:
		return finishReasonToolCalls
	default:
		return finishReasonStop
	}
}

func serverAddress(provider *aiprovider.Provider) string {
	if provider == nil {
		return ""
	}

	return provider.Host()
}

func RecordBusyWait(ctx context.Context, wait time.Duration) {
	aitrace.RecordBusyWait(trace.SpanFromContext(ctx), wait)
}
