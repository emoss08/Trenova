package platformmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Temporal struct {
	metricbase.Base
	activityDuration *prometheus.HistogramVec
	activityTotal    *prometheus.CounterVec
	activityErrors   *prometheus.CounterVec
	workflowDuration *prometheus.HistogramVec
	workflowTotal    *prometheus.CounterVec
	workflowErrors   *prometheus.CounterVec
	activeActivities prometheus.Gauge
	activeWorkflows  prometheus.Gauge
}

func NewTemporal(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Temporal {
	m := &Temporal{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.activityDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "activity_duration_seconds",
			Help:      "Duration of Temporal activity executions in seconds",
			Buckets:   metricbase.TemporalDurationBuckets,
		},
		[]string{"activity_type", "task_queue"},
	)

	m.activityTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "activity_total",
			Help:      "Total number of Temporal activity executions",
		},
		[]string{"activity_type", "task_queue", "status"},
	)

	m.activityErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "activity_errors_total",
			Help:      "Total number of Temporal activity errors",
		},
		[]string{"activity_type", "error_type"},
	)

	m.workflowDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "workflow_duration_seconds",
			Help:      "Duration of Temporal workflow executions in seconds",
			Buckets:   metricbase.TemporalDurationBuckets,
		},
		[]string{"workflow_type"},
	)

	m.workflowTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "workflow_total",
			Help:      "Total number of Temporal workflow executions",
		},
		[]string{"workflow_type", "status"},
	)

	m.workflowErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "workflow_errors_total",
			Help:      "Total number of Temporal workflow errors",
		},
		[]string{"workflow_type", "error_type"},
	)

	m.activeActivities = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "active_activities",
			Help:      "Number of Temporal activities currently being executed",
		},
	)

	m.activeWorkflows = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricbase.Namespace,
			Subsystem: "temporal",
			Name:      "active_workflows",
			Help:      "Number of Temporal workflows currently being executed",
		},
	)

	m.MustRegister(
		m.activityDuration,
		m.activityTotal,
		m.activityErrors,
		m.workflowDuration,
		m.workflowTotal,
		m.workflowErrors,
		m.activeActivities,
		m.activeWorkflows,
	)

	return m
}

func (m *Temporal) RecordActivityExecution(
	activityType, taskQueue string,
	duration float64,
	err error,
) {
	if !m.IsEnabled() {
		return
	}

	status := metricbase.StatusSuccess
	if err != nil {
		status = "error"
		m.activityErrors.WithLabelValues(activityType, classifyError(err)).Inc()
	}

	m.activityTotal.WithLabelValues(activityType, taskQueue, status).Inc()
	m.activityDuration.WithLabelValues(activityType, taskQueue).Observe(duration)
}

func (m *Temporal) RecordWorkflowExecution(workflowType string, duration float64, err error) {
	if !m.IsEnabled() {
		return
	}

	status := metricbase.StatusSuccess
	if err != nil {
		status = "error"
		m.workflowErrors.WithLabelValues(workflowType, classifyError(err)).Inc()
	}

	m.workflowTotal.WithLabelValues(workflowType, status).Inc()
	m.workflowDuration.WithLabelValues(workflowType).Observe(duration)
}

func (m *Temporal) IncrementActiveActivities() {
	m.IfEnabled(func() { m.activeActivities.Inc() })
}

func (m *Temporal) DecrementActiveActivities() {
	m.IfEnabled(func() { m.activeActivities.Dec() })
}

func (m *Temporal) IncrementActiveWorkflows() {
	m.IfEnabled(func() { m.activeWorkflows.Inc() })
}

func (m *Temporal) DecrementActiveWorkflows() {
	m.IfEnabled(func() { m.activeWorkflows.Dec() })
}

func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	return "unknown"
}
