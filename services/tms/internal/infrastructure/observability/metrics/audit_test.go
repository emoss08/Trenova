package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewAudit_Disabled(t *testing.T) {
	t.Parallel()

	m := NewAudit(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
}

func TestNewAudit_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	logger := zap.NewNop()

	m := NewAudit(registry, logger, true)

	require.NotNil(t, m)
	assert.True(t, m.IsEnabled())
	assert.NotNil(t, m.bufferSize)
	assert.NotNil(t, m.dlqSize)
	assert.NotNil(t, m.bufferPushTotal)
	assert.NotNil(t, m.bufferFlushTotal)
	assert.NotNil(t, m.dlqPushTotal)
	assert.NotNil(t, m.dlqRetryTotal)
	assert.NotNil(t, m.directInsertTotal)
	assert.NotNil(t, m.fallbackTotal)
	assert.NotNil(t, m.flushDuration)
	assert.NotNil(t, m.batchSize)
}

func TestAudit_RecordBufferPush(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordBufferPush()
	})
}

func TestAudit_RecordBufferPushFailure(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordBufferPushFailure()
	})
}

func TestAudit_RecordBufferFlush_Success(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordBufferFlush(true, 0.1, 50)
	})
}

func TestAudit_RecordBufferFlush_Failure(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordBufferFlush(false, 0.5, 25)
	})
}

func TestAudit_RecordDLQPush(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordDLQPush(3)
	})
}

func TestAudit_RecordDLQRetry_Success(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordDLQRetry(true)
	})
}

func TestAudit_RecordDLQRetry_Failure(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordDLQRetry(false)
	})
}

func TestAudit_RecordDirectInsert(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordDirectInsert()
	})
}

func TestAudit_RecordFallbackInsert(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordFallbackInsert()
	})
}

func TestAudit_SetBufferSize(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.SetBufferSize(100)
	})
}

func TestAudit_SetDLQSize(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewAudit(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.SetDLQSize(42)
	})
}
