package auditmetrics

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const auditSubsystem = "audit"

type Audit struct {
	metricbase.Base
	entriesWritten   *prometheus.CounterVec
	writeDuration    prometheus.Histogram
	securityEvents   *prometheus.CounterVec
	retentionDeleted *prometheus.CounterVec
	partitionsAhead  prometheus.Gauge
	seals            *prometheus.CounterVec
	verifications    *prometheus.CounterVec
	alerts           *prometheus.CounterVec
	exports          *prometheus.CounterVec
}

func NewAudit(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Audit {
	m := &Audit{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.newWriteMetrics()
	m.newMaintenanceMetrics()

	m.MustRegister(
		m.entriesWritten,
		m.writeDuration,
		m.securityEvents,
		m.retentionDeleted,
		m.partitionsAhead,
		m.seals,
		m.verifications,
		m.alerts,
		m.exports,
	)

	return m
}

func (m *Audit) newWriteMetrics() {
	m.entriesWritten = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "entries_written_total",
		Help:      "Audit entries written, by retention class and outcome of the insert",
	}, []string{metricbase.LabelKind, metricbase.LabelStatus})

	m.writeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "write_duration_seconds",
		Help:      "Time to insert one batch of audit entries",
		Buckets:   []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1},
	})

	m.securityEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "security_event_total",
		Help:      "Total number of security events recorded, by kind and outcome of the write",
	}, []string{metricbase.LabelKind, metricbase.LabelStatus})
}

func (m *Audit) newMaintenanceMetrics() {
	m.retentionDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "retention_deleted_total",
		Help:      "Audit entries deleted by the retention sweep, by retention class",
	}, []string{metricbase.LabelKind})

	m.partitionsAhead = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "partition_months_ahead",
		Help:      "Whole months of audit partitions that exist past the current month",
	})

	m.seals = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "seals_total",
		Help:      "Hourly audit seals written, by outcome",
	}, []string{metricbase.LabelStatus})

	m.verifications = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "verifications_total",
		Help:      "Audit chain verifications, by what they found",
	}, []string{metricbase.LabelOutcome})

	m.alerts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "alerts_total",
		Help:      "Security alerts raised from the audit trail, by kind",
	}, []string{metricbase.LabelKind})

	m.exports = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: auditSubsystem,
		Name:      "exports_total",
		Help:      "Audit log exports finished, by format and outcome",
	}, []string{metricbase.LabelKind, metricbase.LabelStatus})
}

func statusOf(success bool) string {
	if success {
		return metricbase.StatusSuccess
	}
	return metricbase.StatusFailure
}

func (m *Audit) RecordWrite(classes map[string]int, success bool, duration time.Duration) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		status := statusOf(success)
		for class, count := range classes {
			m.entriesWritten.WithLabelValues(class, status).Add(float64(count))
		}
		m.writeDuration.Observe(duration.Seconds())
	})
}

func (m *Audit) RecordSecurityEvent(kind string, success bool) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.securityEvents.WithLabelValues(kind, statusOf(success)).Inc() })
}

func (m *Audit) RecordRetentionDeleted(class string, count int64) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.retentionDeleted.WithLabelValues(class).Add(float64(count)) })
}

func (m *Audit) RecordPartitionMonthsAhead(months int) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.partitionsAhead.Set(float64(months)) })
}

func (m *Audit) RecordSeal(success bool) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.seals.WithLabelValues(statusOf(success)).Inc() })
}

func (m *Audit) RecordVerification(outcome string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.verifications.WithLabelValues(outcome).Inc() })
}

func (m *Audit) RecordAlert(kind string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.alerts.WithLabelValues(kind).Inc() })
}

func (m *Audit) RecordExport(format string, success bool) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.exports.WithLabelValues(format, statusOf(success)).Inc() })
}
