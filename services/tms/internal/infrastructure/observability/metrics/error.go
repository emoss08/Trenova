package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Error struct {
	Base
	errorsTotal     *prometheus.CounterVec
	panicRecoveries prometheus.Counter
}

func NewError(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Error {
	m := &Error{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "errors",
			Name:      "total",
			Help:      "Total number of errors by type and source",
		},
		[]string{"type", "source"},
	)

	m.panicRecoveries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "errors",
			Name:      "panic_recoveries_total",
			Help:      "Total number of recovered panics",
		},
	)

	m.mustRegister(m.errorsTotal, m.panicRecoveries)

	return m
}

func (m *Error) RecordError(errorType, source string) {
	m.ifEnabled(func() { m.errorsTotal.WithLabelValues(errorType, source).Inc() })
}

func (m *Error) RecordPanicRecovery() {
	m.ifEnabled(func() { m.panicRecoveries.Inc() })
}
