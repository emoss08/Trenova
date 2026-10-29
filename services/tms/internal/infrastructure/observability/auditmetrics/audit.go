package auditmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Audit struct {
	metricbase.Base
	bufferSize        prometheus.Gauge
	dlqSize           prometheus.Gauge
	bufferPushTotal   *prometheus.CounterVec
	bufferFlushTotal  *prometheus.CounterVec
	dlqPushTotal      prometheus.Counter
	dlqRetryTotal     *prometheus.CounterVec
	directInsertTotal prometheus.Counter
	fallbackTotal     prometheus.Counter
	criticalBuffered  prometheus.Counter
	securityEvents    *prometheus.CounterVec
	flushDuration     prometheus.Histogram
	batchSize         prometheus.Histogram
}

func NewAudit(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Audit {
	m := &Audit{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.bufferSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "buffer_size",
		Help:      "Current number of entries in the Redis audit buffer",
	})

	m.dlqSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "dlq_size",
		Help:      "Current number of entries in the audit dead-letter queue",
	})

	m.bufferPushTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "buffer_push_total",
		Help:      "Total number of audit entries pushed to buffer",
	}, []string{"status"})

	m.bufferFlushTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "buffer_flush_total",
		Help:      "Total number of audit buffer flushes",
	}, []string{"status"})

	m.dlqPushTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "dlq_push_total",
		Help:      "Total number of audit entries pushed to dead-letter queue",
	})

	m.dlqRetryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "dlq_retry_total",
		Help:      "Total number of DLQ entry retry attempts",
	}, []string{"status"})

	m.directInsertTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "direct_insert_total",
		Help:      "Total number of critical audit entries inserted directly",
	})

	m.fallbackTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "fallback_insert_total",
		Help:      "Total number of audit entries inserted via fallback (buffer push failed)",
	})

	m.criticalBuffered = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "critical_buffered_total",
		Help:      "Total number of critical audit entries buffered because the direct insert failed",
	})

	m.securityEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "security_event_total",
		Help:      "Total number of security events recorded, by kind and outcome of the write",
	}, []string{"kind", "status"})

	m.flushDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "flush_duration_seconds",
		Help:      "Duration of audit buffer flush operations in seconds",
		Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	m.batchSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: "audit",
		Name:      "batch_size",
		Help:      "Size of audit batches processed",
		Buckets:   []float64{1, 10, 50, 100, 250, 500, 1000},
	})

	m.MustRegister(
		m.bufferSize,
		m.dlqSize,
		m.bufferPushTotal,
		m.bufferFlushTotal,
		m.dlqPushTotal,
		m.dlqRetryTotal,
		m.directInsertTotal,
		m.fallbackTotal,
		m.criticalBuffered,
		m.securityEvents,
		m.flushDuration,
		m.batchSize,
	)

	return m
}

func (m *Audit) RecordBufferPush() {
	m.IfEnabled(func() { m.bufferPushTotal.WithLabelValues(metricbase.StatusSuccess).Inc() })
}

func (m *Audit) RecordBufferPushFailure() {
	m.IfEnabled(func() { m.bufferPushTotal.WithLabelValues(metricbase.StatusFailure).Inc() })
}

func (m *Audit) RecordBufferFlush(success bool, duration float64, batchSize int) {
	m.IfEnabled(func() {
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.bufferFlushTotal.WithLabelValues(status).Inc()
		m.flushDuration.Observe(duration)
		m.batchSize.Observe(float64(batchSize))
	})
}

func (m *Audit) RecordDLQPush(count int) {
	m.IfEnabled(func() {
		for range count {
			m.dlqPushTotal.Inc()
		}
	})
}

func (m *Audit) RecordDLQRetry(success bool) {
	m.IfEnabled(func() {
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.dlqRetryTotal.WithLabelValues(status).Inc()
	})
}

func (m *Audit) RecordDirectInsert() {
	m.IfEnabled(func() { m.directInsertTotal.Inc() })
}

func (m *Audit) RecordFallbackInsert() {
	m.IfEnabled(func() { m.fallbackTotal.Inc() })
}

func (m *Audit) SetBufferSize(size int64) {
	m.IfEnabled(func() { m.bufferSize.Set(float64(size)) })
}

func (m *Audit) SetDLQSize(size int64) {
	m.IfEnabled(func() { m.dlqSize.Set(float64(size)) })
}

func (m *Audit) RecordCriticalBuffered(count int) {
	m.IfEnabled(func() { m.criticalBuffered.Add(float64(count)) })
}

func (m *Audit) RecordSecurityEvent(kind string, success bool) {
	m.IfEnabled(func() {
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.securityEvents.WithLabelValues(kind, status).Inc()
	})
}
