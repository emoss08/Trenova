package observability

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestCreateSampler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		samplingRate float64
	}{
		{
			name:         "always sample when rate is 1.0",
			samplingRate: 1.0,
		},
		{
			name:         "always sample when rate exceeds 1.0",
			samplingRate: 1.5,
		},
		{
			name:         "never sample when rate is 0.0",
			samplingRate: 0.0,
		},
		{
			name:         "never sample when rate is negative",
			samplingRate: -0.5,
		},
		{
			name:         "ratio based when rate is 0.5",
			samplingRate: 0.5,
		},
		{
			name:         "ratio based when rate is 0.1",
			samplingRate: 0.1,
		},
		{
			name:         "ratio based when rate is 0.99",
			samplingRate: 0.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.TracingConfig{
				SamplingRate: tt.samplingRate,
			}

			sampler := createSampler(cfg)
			assert.NotNil(t, sampler)
		})
	}
}

func TestGenerateInstanceID(t *testing.T) {
	t.Parallel()

	id := generateInstanceID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "-")

	id2 := generateInstanceID()
	assert.NotEqual(t, id, id2)
}

func TestTracerProvider_IsEnabled_Disabled(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{
		provider: nil,
	}

	assert.False(t, tp.IsEnabled())
}

func TestTracerProvider_Shutdown_NilProvider(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{
		provider: nil,
	}

	err := tp.Shutdown(t.Context())
	assert.NoError(t, err)
}

func TestTracerProvider_AddEvent_NoSpan(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{}

	tp.AddEvent(t.Context(), "test-event",
		attribute.String("key", "value"),
	)
}

func TestTracerProvider_SetAttributes_NoSpan(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{}

	tp.SetAttributes(t.Context(),
		attribute.String("key", "value"),
		attribute.Int("count", 42),
	)
}

func TestTracerProvider_RecordError_NoSpan(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{}

	tp.RecordError(t.Context(), errors.New("test error"))
}

func TestTracerProvider_RecordError_NilError(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{}

	tp.RecordError(t.Context(), nil)
}

func TestTracerProvider_Tracer(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{
		tracer: trace.NewNoopTracerProvider().Tracer("test"),
	}

	tracer := tp.Tracer()
	assert.NotNil(t, tracer)
}

func TestTracerProvider_StartSpan(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{
		tracer: trace.NewNoopTracerProvider().Tracer("test"),
	}

	ctx, span := tp.StartSpan(t.Context(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestCreateExporter_UnsupportedProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.TracingConfig{
		Provider: "unsupported-provider",
	}

	_, err := createExporter(t.Context(), cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported tracing provider")
	assert.Contains(t, err.Error(), "unsupported-provider")
}

func TestCreateExporter_Stdout(t *testing.T) {
	t.Parallel()

	cfg := &config.TracingConfig{
		Provider: "stdout",
	}

	exporter, err := createExporter(t.Context(), cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, exporter)
}

func TestTracerProvider_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ Tracer = (*TracerProvider)(nil)
}
