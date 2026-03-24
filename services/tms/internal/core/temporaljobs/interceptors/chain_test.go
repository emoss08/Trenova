package interceptors

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildWorkerInterceptorChain_BothDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: false,
			},
		},
	}

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: nil,
	})

	assert.Empty(t, chain)
}

func TestBuildWorkerInterceptorChain_LoggingOnly(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: true,
				LogLevel:      "info",
			},
		},
	}

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: nil,
	})

	require.Len(t, chain, 1)
	_, ok := chain[0].(*LoggingInterceptor)
	assert.True(t, ok)
}

func TestBuildWorkerInterceptorChain_MetricsOnly(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: false,
			},
		},
	}

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), true)

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: metricsHandler,
	})

	require.Len(t, chain, 1)
	_, ok := chain[0].(*MetricsInterceptor)
	assert.True(t, ok)
}

func TestBuildWorkerInterceptorChain_BothEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: true,
				LogLevel:      "debug",
			},
		},
	}

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), true)

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: metricsHandler,
	})

	require.Len(t, chain, 2)
	_, ok := chain[0].(*MetricsInterceptor)
	assert.True(t, ok)
	_, ok = chain[1].(*LoggingInterceptor)
	assert.True(t, ok)
}

func TestBuildWorkerInterceptorChain_MetricsDisabledViaHandler(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: false,
			},
		},
	}

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), false)

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: metricsHandler,
	})

	assert.Empty(t, chain)
}

func TestBuildWorkerInterceptorChain_DefaultLogLevel(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: true,
				LogLevel:      "",
			},
		},
	}

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: nil,
	})

	require.Len(t, chain, 1)
	loggingInt, ok := chain[0].(*LoggingInterceptor)
	require.True(t, ok)
	assert.Equal(t, "info", loggingInt.logLevel)
}

func TestNewLoggingInterceptor(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	interceptor := NewLoggingInterceptor(logger, "debug")

	require.NotNil(t, interceptor)
	assert.Equal(t, "debug", interceptor.logLevel)
	assert.NotNil(t, interceptor.logger)
}

func TestNewLoggingInterceptor_InfoLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	interceptor := NewLoggingInterceptor(logger, "info")

	require.NotNil(t, interceptor)
	assert.Equal(t, "info", interceptor.logLevel)
}

func TestNewLoggingInterceptor_EmptyLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	interceptor := NewLoggingInterceptor(logger, "")

	require.NotNil(t, interceptor)
	assert.Equal(t, "", interceptor.logLevel)
}

func TestNewMetricsInterceptor(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), true)
	interceptor := NewMetricsInterceptor(metricsHandler)

	require.NotNil(t, interceptor)
	assert.NotNil(t, interceptor.metrics)
}

func TestNewMetricsInterceptor_DisabledMetrics(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), false)
	interceptor := NewMetricsInterceptor(metricsHandler)

	require.NotNil(t, interceptor)
	assert.NotNil(t, interceptor.metrics)
	assert.False(t, interceptor.metrics.IsEnabled())
}

func TestNewMetricsInterceptor_NilMetrics(t *testing.T) {
	t.Parallel()

	interceptor := NewMetricsInterceptor(nil)

	require.NotNil(t, interceptor)
	assert.Nil(t, interceptor.metrics)
}

func TestBuildWorkerInterceptorChain_OrderMetricsThenLogging(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: true,
				LogLevel:      "warn",
			},
		},
	}

	registry := prometheus.NewRegistry()
	metricsHandler := metrics.NewTemporal(registry, zap.NewNop(), true)

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: metricsHandler,
	})

	require.Len(t, chain, 2)

	_, firstIsMetrics := chain[0].(*MetricsInterceptor)
	assert.True(t, firstIsMetrics)

	_, secondIsLogging := chain[1].(*LoggingInterceptor)
	assert.True(t, secondIsLogging)
}

func TestBuildWorkerInterceptorChain_NilNilMetricsHandler(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Temporal: config.TemporalConfig{
			Interceptors: config.TemporalInterceptorConfig{
				EnableLogging: true,
				LogLevel:      "info",
			},
		},
	}

	chain := BuildWorkerInterceptorChain(ChainParams{
		Config:         cfg,
		Logger:         zap.NewNop(),
		MetricsHandler: nil,
	})

	require.Len(t, chain, 1)
	_, ok := chain[0].(*LoggingInterceptor)
	assert.True(t, ok)
}
