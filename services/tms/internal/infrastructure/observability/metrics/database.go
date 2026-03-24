package metrics

import (
	"strings"

	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Database struct {
	Base
	concurrencyTotal *prometheus.CounterVec
	operatorActions  *prometheus.CounterVec
}

func NewDatabase(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Database {
	m := &Database{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.concurrencyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "db",
			Name:      "concurrency_total",
			Help:      "Total number of database concurrency outcomes by kind, entity, and postgres code",
		},
		[]string{"kind", "entity", "code"},
	)

	m.operatorActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "db",
			Name:      "operator_actions_total",
			Help:      "Total number of database operator actions by action and outcome",
		},
		[]string{"action", "outcome"},
	)

	m.mustRegister(m.concurrencyTotal, m.operatorActions)

	return m
}

func (m *Database) RecordConcurrencyEvent(event dberror.ConcurrencyEvent) {
	m.ifEnabled(func() {
		m.concurrencyTotal.WithLabelValues(
			normalizeMetricLabel(event.Kind),
			normalizeMetricLabel(event.Entity),
			normalizeMetricLabel(event.Code),
		).Inc()
	})
}

func (m *Database) RecordOperatorAction(action, outcome string) {
	m.ifEnabled(func() {
		m.operatorActions.WithLabelValues(
			normalizeMetricLabel(action),
			normalizeMetricLabel(outcome),
		).Inc()
	})
}

func normalizeMetricLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return UnknownValue
	}

	value = strings.ReplaceAll(value, " ", "_")
	return value
}
