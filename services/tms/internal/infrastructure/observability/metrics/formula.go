//revive:disable-next-line:var-naming
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	FormulaTimeoutStageQueued  = "queued"
	FormulaTimeoutStageRunning = "running"

	formulaSubsystem = "formula"
	labelStage       = "stage"
)

type Formula struct {
	Base
	inFlight  prometheus.Gauge
	abandoned prometheus.Gauge
	timeouts  *prometheus.CounterVec
}

func NewFormula(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Formula {
	m := &Formula{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.inFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: formulaSubsystem,
		Name:      "evaluations_in_flight",
		Help:      "Formula evaluations holding a slot, including abandoned ones still running",
	})

	m.abandoned = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: formulaSubsystem,
		Name:      "evaluations_abandoned",
		Help:      "Formula evaluations still running after their caller stopped waiting",
	})

	m.timeouts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: formulaSubsystem,
		Name:      "evaluation_timeouts_total",
		Help:      "Formula evaluations that ran out of time, by whether they were queued or running",
	}, []string{labelStage})

	m.mustRegister(m.inFlight, m.abandoned, m.timeouts)

	return m
}

func (m *Formula) EvaluationStarted() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inFlight.Inc()
}

func (m *Formula) EvaluationFinished() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inFlight.Dec()
}

func (m *Formula) EvaluationAbandoned() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.abandoned.Inc()
}

func (m *Formula) AbandonedEvaluationFinished() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.abandoned.Dec()
}

func (m *Formula) RecordTimeout(stage string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.timeouts.WithLabelValues(stage).Inc()
}
