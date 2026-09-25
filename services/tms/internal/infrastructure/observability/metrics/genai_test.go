//revive:disable-next-line:var-naming
package metrics_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func collect(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()

	var data metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &data))

	out := make(map[string]metricdata.Metrics)
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			out[m.Name] = m
		}
	}

	return out
}

func TestGenAI_RecordsTokenUsageAndDurationBySemconvNames(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	genAI, err := metrics.NewGenAI(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)

	genAI.RecordCall(t.Context(), &metrics.GenAICall{
		Operation:     "chat",
		Provider:      "anthropic",
		RequestModel:  "model-a",
		ResponseModel: "model-a-2026",
		ServerAddress: "api.example.com",
		InputTokens:   1200,
		OutputTokens:  80,
		Duration:      1500 * time.Millisecond,
	})

	recorded := collect(t, reader)
	usage, ok := recorded["gen_ai.client.token.usage"]
	require.True(t, ok)
	assert.Equal(t, "{token}", usage.Unit)
	points := usage.Data.(metricdata.Histogram[int64]).DataPoints
	require.Len(t, points, 2)
	byType := make(map[string]int64, 2)
	for _, point := range points {
		tokenType, _ := point.Attributes.Value(attribute.Key("gen_ai.token.type"))
		byType[tokenType.AsString()] = point.Sum
		provider, _ := point.Attributes.Value(attribute.Key("gen_ai.provider.name"))
		assert.Equal(t, "anthropic", provider.AsString())
	}
	assert.Equal(t, map[string]int64{"input": 1200, "output": 80}, byType)

	duration, ok := recorded["gen_ai.client.operation.duration"]
	require.True(t, ok)
	assert.Equal(t, "s", duration.Unit)
	durations := duration.Data.(metricdata.Histogram[float64]).DataPoints
	require.Len(t, durations, 1)
	assert.InDelta(t, 1.5, durations[0].Sum, 0.0001)
	_, hasError := durations[0].Attributes.Value(attribute.Key("error.type"))
	assert.False(t, hasError)
}

func TestGenAI_AFailedCallCountsItsDurationWithTheErrorAndNoTokens(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	genAI, err := metrics.NewGenAI(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)

	genAI.RecordCall(t.Context(), &metrics.GenAICall{
		Operation: "chat",
		Provider:  "openai",
		ErrorType: "timeout",
		Duration:  30 * time.Second,
	})

	recorded := collect(t, reader)
	_, hasTokens := recorded["gen_ai.client.token.usage"]
	assert.False(t, hasTokens)
	durations := recorded["gen_ai.client.operation.duration"].Data.(metricdata.Histogram[float64]).DataPoints
	require.Len(t, durations, 1)
	errorType, _ := durations[0].Attributes.Value(attribute.Key("error.type"))
	assert.Equal(t, "timeout", errorType.AsString())
}

func TestGenAI_IsSafeWhenMetricsAreOff(t *testing.T) {
	t.Parallel()

	registry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	genAI := registry.GenAI()
	require.NotNil(t, genAI)
	genAI.RecordCall(t.Context(), &metrics.GenAICall{Operation: "chat", Provider: "ollama"})

	var nilRegistry *metrics.Registry
	assert.Nil(t, nilRegistry.GenAI())
	var nilGenAI *metrics.GenAI
	nilGenAI.RecordCall(t.Context(), &metrics.GenAICall{Operation: "chat"})
}
