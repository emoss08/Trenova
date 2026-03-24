package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewError_Disabled(t *testing.T) {
	t.Parallel()

	m := NewError(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
	assert.Nil(t, m.errorsTotal)
	assert.Nil(t, m.panicRecoveries)
}

func TestNewError_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	logger := zap.NewNop()

	m := NewError(registry, logger, true)

	require.NotNil(t, m)
	assert.True(t, m.IsEnabled())
	assert.NotNil(t, m.errorsTotal)
	assert.NotNil(t, m.panicRecoveries)
}

func TestError_RecordError_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewError(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordError("validation", "handler")
	})
}

func TestError_RecordPanicRecovery_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewError(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordPanicRecovery()
	})
}
