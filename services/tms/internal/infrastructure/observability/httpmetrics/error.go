package httpmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Error struct {
	metricbase.Base
	errorsTotal     *prometheus.CounterVec
	panicRecoveries prometheus.Counter
}

func NewError(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Error {
	m := &Error{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "errors",
			Name:                 "total",
			Help:                 "Total number of errors by type and source",
		},
		[]string{"type", "source"},
	)

	m.panicRecoveries = prometheus.NewCounter(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "errors",
			Name:                 "panic_recoveries_total",
			Help:                 "Total number of recovered panics",
		},
	)

	m.MustRegister(m.errorsTotal, m.panicRecoveries)

	return m
}

func (m *Error) RecordError(errorType, source string) {
	m.IfEnabled(func() { m.errorsTotal.WithLabelValues(errorType, source).Inc() })
}

func (m *Error) RecordPanicRecovery() {
	m.IfEnabled(func() { m.panicRecoveries.Inc() })
}
