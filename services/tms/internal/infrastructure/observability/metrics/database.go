//revive:disable-next-line:var-naming
package metrics

import (
	"database/sql"
	"strings"

	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Database struct {
	Base
	concurrencyTotal *prometheus.CounterVec
	operatorActions  *prometheus.CounterVec
	stats            func() sql.DBStats
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

func (m *Database) RegisterSQLStats(stats func() sql.DBStats) {
	m.stats = stats
	m.ifEnabled(func() {
		m.mustRegister(
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "db_pool",
				Name:      "open_connections",
				Help:      "Number of established database connections.",
			}, func() float64 {
				return float64(stats().OpenConnections)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "db_pool",
				Name:      "in_use_connections",
				Help:      "Number of database connections currently in use.",
			}, func() float64 {
				return float64(stats().InUse)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "db_pool",
				Name:      "idle_connections",
				Help:      "Number of idle database connections.",
			}, func() float64 {
				return float64(stats().Idle)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "db_pool",
				Name:      "wait_count_total",
				Help:      "Total number of database connection pool waits.",
			}, func() float64 {
				return float64(stats().WaitCount)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "db_pool",
				Name:      "wait_duration_seconds_total",
				Help:      "Total time spent waiting for database connections.",
			}, func() float64 {
				return stats().WaitDuration.Seconds()
			}),
		)
	})
}

func (m *Database) SQLStats() (sql.DBStats, bool) {
	if m == nil || m.stats == nil {
		return sql.DBStats{}, false
	}
	return m.stats(), true
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
