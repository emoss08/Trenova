//revive:disable-next-line:var-naming
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	labelOp            = "op"
	rulesSubsystem     = "rules"
	ediModuleSubsystem = "edi_module"
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
	Base
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
	m := &WasmModule{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.calls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: subsystem,
		Name:      "calls_total",
		Help:      "Calls into the " + module + " WebAssembly module, by operation and outcome",
	}, []string{labelOp, labelOutcome})

	m.duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: subsystem,
		Name:      "call_duration_seconds",
		Help:      "Time a " + module + " module call took, including the wait for a free instance",
		Buckets:   WasmModuleDurationBuckets,
	}, []string{labelOp})

	m.inUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: subsystem,
		Name:      "instances_in_use",
		Help:      "The " + module + " module's instances running a call",
	})

	m.discarded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: subsystem,
		Name:      "instances_discarded_total",
		Help:      "The " + module + " module's instances thrown away instead of reused, by reason",
	}, []string{labelReason})

	m.mustRegister(m.calls, m.duration, m.inUse, m.discarded)

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
