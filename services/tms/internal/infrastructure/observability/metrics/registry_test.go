package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRegistry_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}
	logger := zap.NewNop()

	registry, err := metrics.NewRegistry(cfg, logger)

	require.NoError(t, err)
	assert.False(t, registry.IsEnabled())
	assert.NotNil(t, registry.HTTP)
	assert.NotNil(t, registry.Error)
	assert.NotNil(t, registry.Database)
	assert.NotNil(t, registry.Temporal)
	assert.NotNil(t, registry.Audit)
}

func TestRegistry_IsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("returns false when disabled", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Monitoring: config.MonitoringConfig{
				Metrics: config.MetricsConfig{
					Enabled: false,
				},
			},
		}

		registry, err := metrics.NewRegistry(cfg, zap.NewNop())
		require.NoError(t, err)

		assert.False(t, registry.IsEnabled())
	})
}

func TestRegistry_Handler_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	handler := registry.Handler()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Equal(t, "Metrics collection is disabled", recorder.Body.String())
}

func TestRegistry_RecordError_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		registry.RecordError("validation", "handler")
	})
}

func TestRegistry_RecordPanicRecovery_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		registry.RecordPanicRecovery()
	})
}

func TestRegistry_IncrementActiveRequests_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		registry.IncrementActiveRequests()
	})
}

func TestRegistry_DecrementActiveRequests_Disabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: false,
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		registry.DecrementActiveRequests()
	})
}

func TestNewRegistry_Enabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
		},
	}
	logger := zap.NewNop()

	registry, err := metrics.NewRegistry(cfg, logger)

	require.NoError(t, err)
	assert.True(t, registry.IsEnabled())
	assert.NotNil(t, registry.Registry())
	assert.NotNil(t, registry.HTTP)
	assert.NotNil(t, registry.Error)
	assert.NotNil(t, registry.Database)
	assert.NotNil(t, registry.Temporal)
	assert.NotNil(t, registry.Audit)
	assert.True(t, registry.HTTP.IsEnabled())
	assert.True(t, registry.Error.IsEnabled())
	assert.True(t, registry.Database.IsEnabled())
	assert.True(t, registry.Temporal.IsEnabled())
	assert.True(t, registry.Audit.IsEnabled())
}

func TestRegistry_Handler_Enabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
		},
	}

	registry, err := metrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	handler := registry.Handler()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "go_")
}
