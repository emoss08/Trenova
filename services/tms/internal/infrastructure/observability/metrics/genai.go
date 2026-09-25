//revive:disable-next-line:var-naming
package metrics

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	genAIMeterName             = "github.com/emoss08/trenova/genai"
	genAITokenUsageName        = "gen_ai.client.token.usage"
	genAIOperationDurationName = "gen_ai.client.operation.duration"
	genAITokenTypeInput        = "input"
	genAITokenTypeOutput       = "output"
)

var (
	genAITokenBuckets = []float64{
		1, 4, 16, 64, 256, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216, 67108864,
	}
	genAIDurationBuckets = []float64{
		0.01, 0.02, 0.04, 0.08, 0.16, 0.32, 0.64, 1.28, 2.56, 5.12, 10.24, 20.48, 40.96, 81.92,
	}
)

type GenAI struct {
	tokens   metric.Int64Histogram
	duration metric.Float64Histogram
}

type GenAICall struct {
	Operation     string
	Provider      string
	RequestModel  string
	ResponseModel string
	ServerAddress string
	ErrorType     string
	InputTokens   int64
	OutputTokens  int64
	Duration      time.Duration
}

type genAIOnce struct {
	once  sync.Once
	genAI *GenAI
}

func (m *Registry) GenAI() *GenAI {
	if m == nil {
		return nil
	}

	m.genAI.once.Do(func() {
		provider, err := m.MeterProvider()
		if err != nil {
			m.logger.Warn("gen_ai client metrics are unavailable", zap.Error(err))
			return
		}

		genAI, err := NewGenAI(provider)
		if err != nil {
			m.logger.Warn("gen_ai client metrics could not be created", zap.Error(err))
			return
		}
		m.genAI.genAI = genAI
	})

	return m.genAI.genAI
}

func NewGenAI(provider metric.MeterProvider) (*GenAI, error) {
	meter := provider.Meter(genAIMeterName)

	tokens, err := meter.Int64Histogram(
		genAITokenUsageName,
		metric.WithUnit("{token}"),
		metric.WithDescription("Number of input and output tokens used"),
		metric.WithExplicitBucketBoundaries(genAITokenBuckets...),
	)
	if err != nil {
		return nil, err
	}

	duration, err := meter.Float64Histogram(
		genAIOperationDurationName,
		metric.WithUnit("s"),
		metric.WithDescription("GenAI operation duration"),
		metric.WithExplicitBucketBoundaries(genAIDurationBuckets...),
	)
	if err != nil {
		return nil, err
	}

	return &GenAI{tokens: tokens, duration: duration}, nil
}

func (g *GenAI) RecordCall(ctx context.Context, call *GenAICall) {
	if g == nil || call == nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, 6)
	attrs = append(attrs,
		attribute.String("gen_ai.operation.name", call.Operation),
		attribute.String("gen_ai.provider.name", call.Provider),
	)
	attrs = appendGenAIString(attrs, "gen_ai.request.model", call.RequestModel)
	attrs = appendGenAIString(attrs, "gen_ai.response.model", call.ResponseModel)
	attrs = appendGenAIString(attrs, "server.address", call.ServerAddress)

	durationAttrs := appendGenAIString(attrs, "error.type", call.ErrorType)
	g.duration.Record(ctx, call.Duration.Seconds(), metric.WithAttributes(durationAttrs...))

	if call.ErrorType != "" && call.InputTokens == 0 && call.OutputTokens == 0 {
		return
	}

	inputAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	inputAttrs = append(inputAttrs, attrs...)
	inputAttrs = append(inputAttrs, attribute.String("gen_ai.token.type", genAITokenTypeInput))
	g.tokens.Record(ctx, call.InputTokens, metric.WithAttributes(inputAttrs...))

	outputAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	outputAttrs = append(outputAttrs, attrs...)
	outputAttrs = append(outputAttrs, attribute.String("gen_ai.token.type", genAITokenTypeOutput))
	g.tokens.Record(ctx, call.OutputTokens, metric.WithAttributes(outputAttrs...))
}

func appendGenAIString(attrs []attribute.KeyValue, key, value string) []attribute.KeyValue {
	if value == "" {
		return attrs
	}

	return append(attrs, attribute.String(key, value))
}
