package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewBase(t *testing.T) {
	t.Parallel()

	t.Run("sets all fields correctly", func(t *testing.T) {
		t.Parallel()
		registry := prometheus.NewRegistry()
		logger := zap.NewNop()

		base := NewBase(registry, logger, true)

		assert.Equal(t, registry, base.registry)
		assert.Equal(t, logger, base.logger)
		assert.True(t, base.enabled)
	})

	t.Run("sets enabled false", func(t *testing.T) {
		t.Parallel()
		base := NewBase(nil, zap.NewNop(), false)

		assert.Nil(t, base.registry)
		assert.False(t, base.enabled)
	})
}

func TestBase_IsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("returns true when enabled", func(t *testing.T) {
		t.Parallel()
		base := NewBase(prometheus.NewRegistry(), zap.NewNop(), true)

		assert.True(t, base.IsEnabled())
	})

	t.Run("returns false when disabled", func(t *testing.T) {
		t.Parallel()
		base := NewBase(nil, zap.NewNop(), false)

		assert.False(t, base.IsEnabled())
	})
}

func TestBase_ifEnabled(t *testing.T) {
	t.Parallel()

	t.Run("calls function when enabled", func(t *testing.T) {
		t.Parallel()
		base := NewBase(prometheus.NewRegistry(), zap.NewNop(), true)

		called := false
		base.ifEnabled(func() { called = true })

		assert.True(t, called)
	})

	t.Run("does not call function when disabled", func(t *testing.T) {
		t.Parallel()
		base := NewBase(nil, zap.NewNop(), false)

		called := false
		base.ifEnabled(func() { called = true })

		assert.False(t, called)
	})
}

func TestBase_mustRegister(t *testing.T) {
	t.Parallel()

	t.Run("registers a counter successfully", func(t *testing.T) {
		t.Parallel()
		registry := prometheus.NewRegistry()
		base := NewBase(registry, zap.NewNop(), true)

		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "test",
			Name:      "counter_total",
		})

		assert.NotPanics(t, func() {
			base.mustRegister(counter)
		})

		metrics, err := registry.Gather()
		assert.NoError(t, err)
		assert.Len(t, metrics, 1)
		assert.Equal(t, "test_counter_total", metrics[0].GetName())
	})
}
