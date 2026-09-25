package completionrouter

import (
	"context"
	"net/url"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
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

type attemptSpec struct {
	operation   string
	provider    *aiprovider.Provider
	attempt     int
	failover    bool
	maxTokens   int
	attribution serviceports.AIUsageAttribution
}

func (s *Service) startAttempt(ctx context.Context, spec attemptSpec) (context.Context, trace.Span) {
	return aitrace.StartAttempt(ctx, &aitrace.AttemptSpec{
		Anchor:        aitrace.ForAttribution(&spec.attribution),
		Operation:     spec.operation,
		ProviderKind:  spec.provider.Kind,
		ProviderID:    spec.provider.ID,
		Model:         spec.provider.Model,
		MaxTokens:     spec.maxTokens,
		ServerAddress: serverAddress(spec.provider),
		Attempt:       spec.attempt,
		Feature:       string(spec.attribution.Feature),
		Attrs:         []attribute.KeyValue{aitrace.AIFailover.Bool(spec.failover)},
	})
}

func (s *Service) settleAttempt(ctx context.Context, span trace.Span, attempt usageAttempt) {
	defer span.End()

	provider := attempt.provider
	cost := attemptCost(attempt)
	if provider != nil && s.health != nil && attempt.err != nil {
		if _, resting := s.health.Resting(provider.ID); resting {
			aitrace.RecordResting(span)
		}
	}

	s.record(ctx, attempt)

	usage := attemptUsage(attempt, cost)
	aitrace.RecordUsage(span, usage)
	errorType := classifyError(attempt.err)
	if attempt.err != nil {
		aitrace.MarkFailed(span, errorType)
	}

	aitrace.TallyFrom(ctx).Attempt(aitrace.AttemptTally{
		ProviderID:   provider.ID.String(),
		ProviderName: aitrace.ProviderName(provider.Kind),
		Model:        usage.ResponseModel,
		Failover:     attempt.failover,
		CostUSD:      cost,
	})

	s.genAI.RecordCall(ctx, &metrics.GenAICall{
		Operation:     attempt.operation,
		Provider:      aitrace.ProviderName(provider.Kind),
		RequestModel:  provider.Model,
		ResponseModel: usage.ResponseModel,
		ServerAddress: serverAddress(provider),
		ErrorType:     errorType,
		InputTokens:   usage.InputTokens,
		OutputTokens:  usage.OutputTokens,
		Duration:      attempt.latency,
	})
}

func attemptCost(attempt usageAttempt) *decimal.Decimal {
	if attempt.outcome == nil || attempt.provider == nil {
		return nil
	}

	return attempt.provider.CostForTask(
		attempt.task,
		attempt.outcome.InputTokens,
		attempt.outcome.OutputTokens,
	)
}

func attemptUsage(attempt usageAttempt, cost *decimal.Decimal) *aitrace.Usage {
	usage := &aitrace.Usage{CostUSD: cost}
	outcome := attempt.outcome
	if outcome == nil {
		return usage
	}

	input := int64(outcome.InputTokens)
	if modeladapter.CacheSeparateFromInput(attempt.provider.Kind) {
		input += int64(outcome.CacheReadTokens + outcome.CacheWriteTokens)
	}
	usage.ResponseModel = outcome.Model
	usage.InputTokens = input
	usage.OutputTokens = int64(outcome.OutputTokens)
	usage.CacheReadTokens = int64(outcome.CacheReadTokens)
	usage.CacheWriteTokens = int64(outcome.CacheWriteTokens)
	usage.ReasoningTokens = int64(outcome.ReasoningTokens)
	usage.Truncated = outcome.Truncated
	if outcome.FinishReason != "" {
		usage.FinishReasons = []string{outcome.FinishReason}
	}

	return usage
}

func finishReason(resp *modeladapter.Response) string {
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

	parsed, err := url.Parse(provider.ResolvedBaseURL())
	if err != nil {
		return ""
	}

	return parsed.Hostname()
}

func recordBusyWait(ctx context.Context, wait time.Duration) {
	aitrace.RecordBusyWait(trace.SpanFromContext(ctx), wait)
}
