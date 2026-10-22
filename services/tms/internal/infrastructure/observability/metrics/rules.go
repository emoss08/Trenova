//revive:disable-next-line:var-naming
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	rulesSubsystem = "rules"
	labelOp        = "op"
)

var RulesDurationBuckets = []float64{
	.0001,
	.00025,
	.0005,
	.001,
	.0025,
	.005,
	.01,
	.025,
	.05,
	.1,
	.25,
	1,
}

type Rules struct {
	Base
	calls     *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	inUse     prometheus.Gauge
	discarded *prometheus.CounterVec
}

func NewRules(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Rules {
	m := &Rules{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.calls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: rulesSubsystem,
		Name:      "calls_total",
		Help:      "Calls into the rules WebAssembly module, by operation and outcome",
	}, []string{labelOp, labelOutcome})

	m.duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: rulesSubsystem,
		Name:      "call_duration_seconds",
		Help:      "Time a rules call took, including the wait for a free instance",
		Buckets:   RulesDurationBuckets,
	}, []string{labelOp})

	m.inUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: rulesSubsystem,
		Name:      "instances_in_use",
		Help:      "Rules module instances running a call",
	})

	m.discarded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: rulesSubsystem,
		Name:      "instances_discarded_total",
		Help:      "Rules module instances thrown away instead of reused, by reason",
	}, []string{labelReason})

	m.mustRegister(m.calls, m.duration, m.inUse, m.discarded)

	return m
}

func (m *Rules) CallFinished(op, outcome string, duration time.Duration) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.calls.WithLabelValues(op, outcome).Inc()
	m.duration.WithLabelValues(op).Observe(duration.Seconds())
}

func (m *Rules) InstanceAcquired() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inUse.Inc()
}

func (m *Rules) InstanceReleased() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inUse.Dec()
}

func (m *Rules) InstanceDiscarded(reason string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.discarded.WithLabelValues(reason).Inc()
}
