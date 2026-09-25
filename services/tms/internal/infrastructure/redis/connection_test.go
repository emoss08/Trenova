package redis

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func unreachableCache(tracing bool) *config.Config {
	cfg := &config.Config{}
	cfg.Cache = config.CacheConfig{
		Host:        "127.0.0.1",
		Port:        1,
		DialTimeout: 100 * time.Millisecond,
		MaxRetries:  -1,
	}
	cfg.Monitoring.Tracing.Enabled = tracing

	return cfg
}

func TestNewConnection_TracesCommandsOnceTheTracerStarts(t *testing.T) {
	client, err := NewConnection(ConnectionParams{
		Config: unreachableCache(true),
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })
	otel.SetTracerProvider(provider)

	require.Error(t, client.Ping(t.Context()).Err())

	names := make([]string, 0, len(recorder.Ended()))
	for _, span := range recorder.Ended() {
		names = append(names, span.Name())
	}
	assert.Contains(t, names, "ping",
		"a client built before the tracer provider started must still be traced")
}

func TestNewConnection_LeavesCommandsUntracedWhenTracingIsOff(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })
	otel.SetTracerProvider(provider)

	client, err := NewConnection(ConnectionParams{
		Config: unreachableCache(false),
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	require.Error(t, client.Ping(t.Context()).Err())

	assert.Empty(t, recorder.Ended())
}
