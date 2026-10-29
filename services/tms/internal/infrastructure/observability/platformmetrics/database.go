package platformmetrics

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	dbCacheSubsystem = "db_cache"
	rlsLabelPool     = "pool"
	rlsLabelEvent    = "event"
	rlsLabelOutcome  = "outcome"
)

type Database struct {
	metricbase.Base
	concurrencyTotal *prometheus.CounterVec
	operatorActions  *prometheus.CounterVec
	rlsScopeEvents   *prometheus.CounterVec
	slowWaits        *prometheus.CounterVec
	stats            func() PoolStats
}

func NewDatabase(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Database {
	m := &Database{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.concurrencyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "db",
			Name:                 "concurrency_total",
			Help:                 "Total number of database concurrency outcomes by kind, entity, and postgres code",
		},
		[]string{"kind", "entity", "code"},
	)

	m.operatorActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "db",
			Name:                 "operator_actions_total",
			Help:                 "Total number of database operator actions by action and outcome",
		},
		[]string{"action", "outcome"},
	)

	m.rlsScopeEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "db",
			Name:                 "rls_scope_events_total",
			Help:                 "Total number of row-level security scope decisions by pool, event, and outcome",
		},
		[]string{rlsLabelPool, rlsLabelEvent, rlsLabelOutcome},
	)

	m.slowWaits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "db",
			Name:                 "slow_waits_total",
			Help:                 "Total number of waits for a tenant slot, a connection or a cache fill past database.slowWaitWarning, by component, pool and kind",
		},
		[]string{"component", "pool", "kind"},
	)

	m.MustRegister(m.concurrencyTotal, m.operatorActions, m.rlsScopeEvents, m.slowWaits)

	return m
}

type PoolStats struct {
	Open         int
	InUse        int
	Idle         int
	MaxOpen      int
	WaitCount    int64
	WaitDuration time.Duration
}

func (m *Database) RegisterPoolStats(stats func() PoolStats) {
	m.stats = stats
	m.IfEnabled(func() {
		m.MustRegister(
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "open_connections",
				Help:                 "Number of established database connections.",
			}, func() float64 {
				return float64(stats().Open)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "in_use_connections",
				Help:                 "Number of database connections currently in use.",
			}, func() float64 {
				return float64(stats().InUse)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "idle_connections",
				Help:                 "Number of idle database connections.",
			}, func() float64 {
				return float64(stats().Idle)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "wait_count_total",
				Help:                 "Total number of database connection pool waits.",
			}, func() float64 {
				return float64(stats().WaitCount)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "wait_duration_seconds_total",
				Help:                 "Total time spent waiting for database connections.",
			}, func() float64 {
				return stats().WaitDuration.Seconds()
			}),
		)
	})
}

type ResultCacheStats struct {
	Hits      int64
	Misses    int64
	Bypasses  int64
	Errors    int64
	Bytes     int64
	Versions  int
	Listening bool
}

type RuntimeStats struct {
	Cache              ResultCacheStats
	TenantWaits        int64
	ReadCommitFailures int64
}

func (m *Database) RegisterRuntimeStats(stats func() RuntimeStats) {
	m.IfEnabled(func() {
		m.MustRegister(
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "hits_total",
				Help:                 "Total number of database reads served from the result cache.",
			}, func() float64 {
				return float64(stats().Cache.Hits)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "misses_total",
				Help:                 "Total number of cacheable database reads that read the database.",
			}, func() float64 {
				return float64(stats().Cache.Misses)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "bypasses_total",
				Help:                 "Total number of cacheable database reads where the result cache did not apply.",
			}, func() float64 {
				return float64(stats().Cache.Bypasses)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "errors_total",
				Help:                 "Total number of result cache store, listener and decoder failures.",
			}, func() float64 {
				return float64(stats().Cache.Errors)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "bytes",
				Help:                 "Bytes held by the in-process result cache.",
			}, func() float64 {
				return float64(stats().Cache.Bytes)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "versions",
				Help:                 "Number of (tenant, table) data versions the result cache holds.",
			}, func() float64 {
				return float64(stats().Cache.Versions)
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            dbCacheSubsystem,
				Name:                 "listening",
				Help:                 "1 while the result cache's data version listener is connected; the cache is bypassed at 0.",
			}, func() float64 {
				if stats().Cache.Listening {
					return 1
				}
				return 0
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "tenant_waits_total",
				Help:                 "Total number of times work waited for one of its tenant's concurrency slots.",
			}, func() float64 {
				return float64(stats().TenantWaits)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				metricbase.Namespace: metricbase.Namespace,
				Subsystem:            "db_pool",
				Name:                 "read_commit_failures_total",
				Help:                 "Total number of connections closed because the reply to a read transaction's end failed.",
			}, func() float64 {
				return float64(stats().ReadCommitFailures)
			}),
		)
	})
}

func (m *Database) PoolStats() (PoolStats, bool) {
	if m == nil || m.stats == nil {
		return PoolStats{}, false
	}
	return m.stats(), true
}

func PoolStatsFields(stats PoolStats) []zap.Field {
	return []zap.Field{
		zap.Int("db_pool_open", stats.Open),
		zap.Int("db_pool_in_use", stats.InUse),
		zap.Int("db_pool_idle", stats.Idle),
		zap.Int64("db_pool_wait_count", stats.WaitCount),
		zap.Duration("db_pool_wait_duration", stats.WaitDuration),
		zap.Int("db_pool_max_open", stats.MaxOpen),
	}
}

func (m *Database) RecordConcurrencyEvent(event dberror.ConcurrencyEvent) {
	m.IfEnabled(func() {
		m.concurrencyTotal.WithLabelValues(
			normalizeMetricLabel(event.Kind),
			normalizeMetricLabel(event.Entity),
			normalizeMetricLabel(event.Code),
		).Inc()
	})
}

func (m *Database) RecordOperatorAction(action, outcome string) {
	m.IfEnabled(func() {
		m.operatorActions.WithLabelValues(
			normalizeMetricLabel(action),
			normalizeMetricLabel(outcome),
		).Inc()
	})
}

func (m *Database) RecordRLSScopeEvent(pool, event, outcome string) {
	m.IfEnabled(func() {
		m.rlsScopeEvents.WithLabelValues(
			normalizeMetricLabel(pool),
			normalizeMetricLabel(event),
			normalizeMetricLabel(outcome),
		).Inc()
	})
}

func (m *Database) RecordSlowWait(component, pool, kind string) {
	m.IfEnabled(func() {
		m.slowWaits.WithLabelValues(
			normalizeMetricLabel(component),
			normalizeMetricLabel(pool),
			normalizeMetricLabel(kind),
		).Inc()
	})
}

func normalizeMetricLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return metricbase.UnknownValue
	}

	value = strings.ReplaceAll(value, " ", "_")
	return value
}
