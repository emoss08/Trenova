package platformmetrics

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	labelOp            = "op"
	RulesSubsystem     = "rules"
	EDIModuleSubsystem = "edi_module"
)

var WasmModuleDurationBuckets = []float64{
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

type WasmModule struct {
	metricbase.Base
	calls     *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	inUse     prometheus.Gauge
	discarded *prometheus.CounterVec
}

func NewWasmModule(
	registry *prometheus.Registry,
	logger *zap.Logger,
	enabled bool,
	subsystem, module string,
) *WasmModule {
	m := &WasmModule{Base: metricbase.NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.calls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: subsystem,
		Name:      "calls_total",
		Help:      "Calls into the " + module + " WebAssembly module, by operation and outcome",
	}, []string{labelOp, metricbase.LabelOutcome})

	m.duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: subsystem,
		Name:      "call_duration_seconds",
		Help:      "Time a " + module + " module call took, including the wait for a free instance",
		Buckets:   WasmModuleDurationBuckets,
	}, []string{labelOp})

	m.inUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricbase.Namespace,
		Subsystem: subsystem,
		Name:      "instances_in_use",
		Help:      "The " + module + " module's instances running a call",
	})

	m.discarded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: subsystem,
		Name:      "instances_discarded_total",
		Help:      "The " + module + " module's instances thrown away instead of reused, by reason",
	}, []string{metricbase.LabelReason})

	m.MustRegister(m.calls, m.duration, m.inUse, m.discarded)

	return m
}

func (m *WasmModule) CallFinished(op, outcome string, duration time.Duration) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.calls.WithLabelValues(op, outcome).Inc()
	m.duration.WithLabelValues(op).Observe(duration.Seconds())
}

func (m *WasmModule) InstanceAcquired() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inUse.Inc()
}

func (m *WasmModule) InstanceReleased() {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.inUse.Dec()
}

func (m *WasmModule) InstanceDiscarded(reason string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.discarded.WithLabelValues(reason).Inc()
}
