package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	permissionSubsystem = "permission"

	permissionLabelOutcome = "outcome"
	permissionLabelPath    = "path"

	PermissionCacheHit   = "hit"
	PermissionCacheMiss  = "miss"
	PermissionCacheError = "error"

	PermissionPathCached     = "cached"
	PermissionPathEffective  = "effective_permissions"
	PermissionPathSimulation = "simulation"
)

type Permission struct {
	Base
	cacheLookupsTotal   *prometheus.CounterVec
	computeTotal        *prometheus.CounterVec
	computeDuration     *prometheus.HistogramVec
	cacheLookupDuration prometheus.Histogram
	closureRoles        prometheus.Histogram
}

func NewPermission(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Permission {
	m := &Permission{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.cacheLookupsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: permissionSubsystem,
		Name:      "cache_lookups_total",
		Help:      "Permission cache lookups by outcome (hit, miss, error)",
	}, []string{permissionLabelOutcome})

	m.computeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: permissionSubsystem,
		Name:      "computes_total",
		Help: "Permission recomputations by the path that asked for them " +
			"(cached, effective_permissions, simulation)",
	}, []string{permissionLabelPath})

	m.computeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: permissionSubsystem,
		Name:      "compute_duration_seconds",
		Help:      "Duration of a permission recomputation, by the path that asked for it",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{permissionLabelPath})

	m.cacheLookupDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: permissionSubsystem,
		Name:      "cache_lookup_duration_seconds",
		Help:      "Duration of a permission cache lookup",
		Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1},
	})

	m.closureRoles = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: permissionSubsystem,
		Name:      "closure_roles",
		Help: "Roles returned by one inheritance closure, including inherited ones; " +
			"the cost of a recomputation scales with this",
		Buckets: []float64{1, 2, 3, 5, 8, 13, 21, 34, 55},
	})

	m.mustRegister(
		m.cacheLookupsTotal,
		m.computeTotal,
		m.computeDuration,
		m.cacheLookupDuration,
		m.closureRoles,
	)

	return m
}

func (m *Permission) RecordCacheLookup(outcome string, duration time.Duration) {
	m.ifEnabled(func() {
		m.cacheLookupsTotal.WithLabelValues(outcome).Inc()
		if duration > 0 {
			m.cacheLookupDuration.Observe(duration.Seconds())
		}
	})
}

func (m *Permission) RecordCompute(path string, duration time.Duration) {
	m.ifEnabled(func() {
		m.computeTotal.WithLabelValues(path).Inc()
		m.computeDuration.WithLabelValues(path).Observe(duration.Seconds())
	})
}

func (m *Permission) RecordClosureSize(roleCount int) {
	if roleCount <= 0 {
		return
	}

	m.ifEnabled(func() {
		m.closureRoles.Observe(float64(roleCount))
	})
}
