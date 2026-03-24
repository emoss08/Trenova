package metrics

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewTemporal_Disabled(t *testing.T) {
	t.Parallel()

	m := NewTemporal(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
}

func TestNewTemporal_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	logger := zap.NewNop()

	m := NewTemporal(registry, logger, true)

	require.NotNil(t, m)
	assert.True(t, m.IsEnabled())
}

func TestTemporal_RecordActivityExecution_Success(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordActivityExecution("send-email", "default", 0.5, nil)
	})
}

func TestTemporal_RecordActivityExecution_Error(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordActivityExecution("send-email", "default", 1.2, errors.New("timeout"))
	})
}

func TestTemporal_RecordWorkflowExecution_Success(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordWorkflowExecution("order-processing", 2.5, nil)
	})
}

func TestTemporal_RecordWorkflowExecution_Error(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordWorkflowExecution("order-processing", 3.0, errors.New("failed"))
	})
}

func TestTemporal_IncrementDecrementActiveActivities(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.IncrementActiveActivities()
		m.IncrementActiveActivities()
		m.DecrementActiveActivities()
	})
}

func TestTemporal_IncrementDecrementActiveWorkflows(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewTemporal(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.IncrementActiveWorkflows()
		m.IncrementActiveWorkflows()
		m.DecrementActiveWorkflows()
	})
}

func TestTemporal_RecordActivityExecution_Disabled(t *testing.T) {
	t.Parallel()

	m := NewTemporal(nil, zap.NewNop(), false)

	assert.NotPanics(t, func() {
		m.RecordActivityExecution("send-email", "default", 0.5, nil)
	})
}

func TestClassifyError_Nil(t *testing.T) {
	t.Parallel()

	result := classifyError(nil)
	assert.Equal(t, "none", result)
}

func TestClassifyError_NonNil(t *testing.T) {
	t.Parallel()

	result := classifyError(errors.New("something went wrong"))
	assert.Equal(t, "unknown", result)
}
