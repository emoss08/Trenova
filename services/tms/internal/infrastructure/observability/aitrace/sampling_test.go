package aitrace

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func localProvider(defaultRate float64) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	recorder := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(recorder),
		sdktrace.WithIDGenerator(NewIDGenerator()),
		sdktrace.WithSampler(NewSampler(defaultRate)),
	)

	return provider, recorder
}

func useSamplingRate(t *testing.T, rate float64) {
	t.Helper()

	SetSamplingRate(rate)
	t.Cleanup(func() { SetSamplingRate(defaultAISamplingRate) })
}

func TestSamplingRate_DefaultsToEverything(t *testing.T) {
	assert.InDelta(t, 1.0, SamplingRate(), 0)
	assert.True(t, AnchorFor(AnchorAgentRun, "ar_default").Sampled)
}

func TestSampler_DecisionEqualsAnchorSampledAtEveryRate(t *testing.T) {
	for _, rate := range []float64{0, 0.1, 0.25, 0.5, 0.75, 1} {
		useSamplingRate(t, rate)
		ratio := sdktrace.TraceIDRatioBased(rate)
		provider, recorder := localProvider(0)
		sampled := 0

		for i := range 400 {
			a := AnchorFor(AnchorAssistantTurn, "atrn_"+strconv.Itoa(i))
			want := ratio.ShouldSample(sdktrace.SamplingParameters{
				ParentContext: t.Context(),
				TraceID:       a.TraceID,
			}).Decision == sdktrace.RecordAndSample
			require.Equal(t, want, a.Sampled, "rate %v anchor %d", rate, i)

			root := NewSampler(0).ShouldSample(sdktrace.SamplingParameters{
				ParentContext: withForcedAnchor(t.Context(), a),
				TraceID:       a.TraceID,
				Name:          "invoke_agent",
			})
			assert.Equal(t, a.Sampled, root.Decision == sdktrace.RecordAndSample,
				"the anchored root follows the anchor at rate %v", rate)

			_, child := provider.Tracer("test").Start(ContextWithAnchor(t.Context(), a), "child")
			assert.Equal(t, a.Sampled, child.SpanContext().IsSampled(),
				"a span under the anchor follows it at rate %v", rate)
			child.End()

			if a.Sampled {
				sampled++
			}
		}

		assert.Len(t, recorder.GetSpans(), sampled)
		switch rate {
		case 0:
			assert.Zero(t, sampled)
		case 1:
			assert.Equal(t, 400, sampled)
		default:
			assert.InDelta(t, rate*400, float64(sampled), 60, "rate %v", rate)
		}
	}
}

func TestSampler_UnanchoredRootsUseTheDefaultRate(t *testing.T) {
	t.Parallel()

	params := sdktrace.SamplingParameters{
		ParentContext: t.Context(),
		TraceID:       randomTraceID(),
		Name:          "GET /shipments",
	}

	assert.Equal(t, sdktrace.Drop, NewSampler(0).ShouldSample(params).Decision)
	assert.Equal(t, sdktrace.RecordAndSample, NewSampler(1).ShouldSample(params).Decision)
}

func TestSampler_AForcedAnchorOnlyAppliesToItsOwnTrace(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_forced_other")
	a.Sampled = true

	result := NewSampler(0).ShouldSample(sdktrace.SamplingParameters{
		ParentContext: withForcedAnchor(t.Context(), a),
		TraceID:       randomTraceID(),
	})

	assert.Equal(t, sdktrace.Drop, result.Decision)
}

func TestSampler_FollowsTheParentWhateverTheRate(t *testing.T) {
	t.Parallel()

	sampled := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    randomTraceID(),
		SpanID:     randomSpanID(),
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	unsampled := sampled.WithTraceFlags(0)

	withSampled := NewSampler(0).ShouldSample(sdktrace.SamplingParameters{
		ParentContext: trace.ContextWithRemoteSpanContext(t.Context(), sampled),
		TraceID:       sampled.TraceID(),
	})
	withUnsampled := NewSampler(1).ShouldSample(sdktrace.SamplingParameters{
		ParentContext: trace.ContextWithRemoteSpanContext(t.Context(), unsampled),
		TraceID:       unsampled.TraceID(),
	})

	assert.Equal(t, sdktrace.RecordAndSample, withSampled.Decision)
	assert.Equal(t, sdktrace.Drop, withUnsampled.Decision)
}

func TestRatioSampler_ClampsOutOfRangeRates(t *testing.T) {
	t.Parallel()

	params := sdktrace.SamplingParameters{ParentContext: t.Context(), TraceID: randomTraceID()}

	assert.Equal(t, sdktrace.RecordAndSample, RatioSampler(1.5).ShouldSample(params).Decision)
	assert.Equal(t, sdktrace.Drop, RatioSampler(-0.5).ShouldSample(params).Decision)
	assert.Contains(t, NewSampler(0.5).Description(), "AIAnchorAware")
}
