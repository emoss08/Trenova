//revive:disable-next-line:var-naming
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// AIAudit measures the AI audit trail: how far behind its sources the
// projector is, what it writes, what verification finds and how exports end.
type AIAudit struct {
	Base
	projectorLag   *prometheus.GaugeVec
	projectedTotal prometheus.Counter
	passTotal      *prometheus.CounterVec
	verifications  *prometheus.CounterVec
	exports        *prometheus.CounterVec
	prunedTotal    prometheus.Counter
}

func NewAIAudit(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *AIAudit {
	m := &AIAudit{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.projectorLag = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "projector_lag_seconds",
		Help:      "Age of the oldest source row the AI audit projector has not yet read, per source",
	}, []string{"source"})

	m.projectedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "events_projected_total",
		Help:      "AI audit events appended to the trail",
	})

	m.passTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "projector_passes_total",
		Help:      "AI audit projector passes by result",
	}, []string{"result"})

	m.verifications = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "verifications_total",
		Help:      "AI audit chain verifications by status",
	}, []string{"status"})

	m.exports = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "exports_total",
		Help:      "AI audit exports by format and final status",
	}, []string{"format", "status"})

	m.prunedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai_audit",
		Name:      "events_pruned_total",
		Help:      "AI audit events removed by the retention sweep",
	})

	m.mustRegister(
		m.projectorLag,
		m.projectedTotal,
		m.passTotal,
		m.verifications,
		m.exports,
		m.prunedTotal,
	)

	return m
}

func (m *AIAudit) RecordLag(source string, seconds float64) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.projectorLag.WithLabelValues(source).Set(seconds)
}

func (m *AIAudit) RecordProjected(events int) {
	if m == nil || !m.IsEnabled() || events <= 0 {
		return
	}
	m.projectedTotal.Add(float64(events))
}

func (m *AIAudit) RecordPass(result string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.passTotal.WithLabelValues(result).Inc()
}

func (m *AIAudit) RecordVerification(status string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.verifications.WithLabelValues(status).Inc()
}

func (m *AIAudit) RecordExport(format, status string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.exports.WithLabelValues(format, status).Inc()
}

func (m *AIAudit) RecordPruned(events int) {
	if m == nil || !m.IsEnabled() || events <= 0 {
		return
	}
	m.prunedTotal.Add(float64(events))
}
