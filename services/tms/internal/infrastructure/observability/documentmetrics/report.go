package documentmetrics

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	reportingSubsystem = "reporting"
	reportLabelStatus  = "status"
	reportLabelFormat  = "format"
	reportLabelOutcome = "outcome"
)

type Report struct {
	metricbase.Base
	runsTotal            *prometheus.CounterVec
	runDurationSeconds   *prometheus.HistogramVec
	runRows              prometheus.Histogram
	runBytes             prometheus.Histogram
	enqueueRejections    *prometheus.CounterVec
	compileErrorsTotal   *prometheus.CounterVec
	previewTotal         *prometheus.CounterVec
	previewDuration      prometheus.Histogram
	cacheLookupsTotal    *prometheus.CounterVec
	artifactsCleaned     prometheus.Counter
	zombieRunsReconciled prometheus.Counter
	deliveriesTotal      *prometheus.CounterVec
}

func NewReport(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Report {
	m := &Report{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.registerRunMetrics()
	m.registerOperationalMetrics()

	m.MustRegister(
		m.runsTotal,
		m.runDurationSeconds,
		m.runRows,
		m.runBytes,
		m.enqueueRejections,
		m.compileErrorsTotal,
		m.previewTotal,
		m.previewDuration,
		m.cacheLookupsTotal,
		m.artifactsCleaned,
		m.zombieRunsReconciled,
		m.deliveriesTotal,
	)

	return m
}

func (m *Report) registerRunMetrics() {
	m.runsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "runs_total",
		Help:                 "Total number of report runs by terminal status, format, and trigger",
	}, []string{reportLabelStatus, reportLabelFormat, "trigger"})

	m.runDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "run_duration_seconds",
		Help:                 "Wall-clock duration of report runs",
		Buckets:              []float64{1, 5, 15, 30, 60, 120, 300, 600, 1200, 1800},
	}, []string{reportLabelFormat})

	m.runRows = prometheus.NewHistogram(prometheus.HistogramOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "run_rows",
		Help:                 "Rows produced per report run",
		Buckets:              prometheus.ExponentialBuckets(10, 10, 7),
	})

	m.runBytes = prometheus.NewHistogram(prometheus.HistogramOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "run_bytes",
		Help:                 "Artifact size per report run in bytes",
		Buckets:              prometheus.ExponentialBuckets(1024, 8, 8),
	})
}

func (m *Report) registerOperationalMetrics() {
	m.enqueueRejections = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "enqueue_rejections_total",
		Help:                 "Report run enqueue rejections by reason (concurrency, queue_depth)",
	}, []string{metricbase.LabelReason})

	m.compileErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "compile_errors_total",
		Help:                 "Report compile failures by stage (validation, authorization, cost)",
	}, []string{"stage"})

	m.previewTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "preview_total",
		Help:                 "Builder preview executions by outcome",
	}, []string{reportLabelOutcome})

	m.previewDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "preview_duration_seconds",
		Help:                 "Duration of builder preview executions",
		Buckets:              []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	})

	m.cacheLookupsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "cache_lookups_total",
		Help:                 "Report result cache lookups by outcome (hit, miss, bypass)",
	}, []string{reportLabelOutcome})

	m.artifactsCleaned = prometheus.NewCounter(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "artifacts_cleaned_total",
		Help:                 "Expired report artifacts deleted by the cleanup schedule",
	})

	m.zombieRunsReconciled = prometheus.NewCounter(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "zombie_runs_reconciled_total",
		Help:                 "Abandoned report runs marked failed by the reconciliation schedule",
	})

	m.deliveriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		metricbase.Namespace: metricbase.Namespace,
		Subsystem:            reportingSubsystem,
		Name:                 "deliveries_total",
		Help:                 "Scheduled report deliveries by channel (email, notification) and outcome",
	}, []string{"channel", reportLabelOutcome})
}

func (m *Report) RecordRun(
	status, format, trigger string,
	duration time.Duration,
	rows, bytes int64,
) {
	m.IfEnabled(func() {
		m.runsTotal.WithLabelValues(status, format, trigger).Inc()
		m.runDurationSeconds.WithLabelValues(format).Observe(duration.Seconds())
		if rows > 0 {
			m.runRows.Observe(float64(rows))
		}
		if bytes > 0 {
			m.runBytes.Observe(float64(bytes))
		}
	})
}

func (m *Report) RecordEnqueueRejection(reason string) {
	m.IfEnabled(func() {
		m.enqueueRejections.WithLabelValues(reason).Inc()
	})
}

func (m *Report) RecordCompileError(stage string) {
	m.IfEnabled(func() {
		m.compileErrorsTotal.WithLabelValues(stage).Inc()
	})
}

func (m *Report) RecordPreview(outcome string, duration time.Duration) {
	m.IfEnabled(func() {
		m.previewTotal.WithLabelValues(outcome).Inc()
		m.previewDuration.Observe(duration.Seconds())
	})
}

func (m *Report) RecordCacheLookup(outcome string) {
	m.IfEnabled(func() {
		m.cacheLookupsTotal.WithLabelValues(outcome).Inc()
	})
}

func (m *Report) RecordArtifactsCleaned(count int) {
	m.IfEnabled(func() {
		m.artifactsCleaned.Add(float64(count))
	})
}

func (m *Report) RecordZombieRunsReconciled(count int) {
	m.IfEnabled(func() {
		m.zombieRunsReconciled.Add(float64(count))
	})
}

func (m *Report) RecordDelivery(channel, outcome string) {
	m.IfEnabled(func() {
		m.deliveriesTotal.WithLabelValues(channel, outcome).Inc()
	})
}
